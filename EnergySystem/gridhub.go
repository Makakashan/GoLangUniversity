package main

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type GridHub struct {
	ozeSources         []EnergySource
	curtailableSources []CurtailableSource
	conventional       *ConventionalPlant
	ess                EnergyStorage
	consumers          []Consumer
	supplyChans        map[string]chan SupplyStatus
	demandChan         chan DemandReport
	forecastChan       <-chan ForecastReport
	logger             DataLogger

	currentDemands   map[string]float64
	sheddedConsumers map[string]bool
	gridStep         int

	mu            sync.RWMutex
	statsMu       sync.Mutex
	loadShedCount int
}

func NewGridHub(forecastChan <-chan ForecastReport, logger DataLogger) *GridHub {
	return &GridHub{
		ozeSources:         make([]EnergySource, 0),
		curtailableSources: make([]CurtailableSource, 0),
		consumers:          make([]Consumer, 0),
		supplyChans:        make(map[string]chan SupplyStatus),
		demandChan:         make(chan DemandReport, 100),
		forecastChan:       forecastChan,
		logger:             logger,
		currentDemands:     make(map[string]float64),
		sheddedConsumers:   make(map[string]bool),
		gridStep:           0,
		loadShedCount:      0,
	}
}

func (gh *GridHub) AddOZESource(source EnergySource) {
	gh.mu.Lock()
	gh.ozeSources = append(gh.ozeSources, source)
	if curtailable, ok := source.(CurtailableSource); ok {
		gh.curtailableSources = append(gh.curtailableSources, curtailable)
	}
	gh.mu.Unlock()
}

func (gh *GridHub) SetConventional(plant *ConventionalPlant) {
	gh.conventional = plant
}

func (gh *GridHub) SetESS(ess EnergyStorage) {
	gh.ess = ess
}

func (gh *GridHub) RegisterConsumer(consumer Consumer) <-chan SupplyStatus {
	gh.mu.Lock()
	defer gh.mu.Unlock()

	gh.consumers = append(gh.consumers, consumer)
	statusChan := make(chan SupplyStatus, 10)
	gh.supplyChans[consumer.GetID()] = statusChan
	return statusChan
}

func (gh *GridHub) AddConsumer(consumer Consumer) {
	gh.RegisterConsumer(consumer)
}

func (gh *GridHub) GetDemandChan() chan<- DemandReport {
	return gh.demandChan
}

func (gh *GridHub) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(GridStep)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("[GridHub] Zamykanie...")
			return

		case demand := <-gh.demandChan:
			gh.mu.Lock()
			gh.currentDemands[demand.ID] = demand.DemandMW
			gh.mu.Unlock()

		case forecast := <-gh.forecastChan:
			gh.handleForecast(forecast)

		case <-ticker.C:
			gh.performBalance()
			gh.gridStep++
		}
	}
}

func (gh *GridHub) handleForecast(forecast ForecastReport) {
	if forecast.OZEChangePct < -10 && forecast.Confidence > 0.5 {
		if gh.conventional != nil && gh.conventional.GetStatus() == "Off" {
			fmt.Printf("[GridHub] Prognoza spadku OZE o %.1f%% — uruchamiam elektrownię węglową\n",
				forecast.OZEChangePct)
			gh.conventional.Start()
		}
	}

	if forecast.OZEChangePct > 20 && forecast.Confidence > 0.7 {
		if gh.conventional != nil && gh.conventional.GetStatus() == "Running" {
			fmt.Printf("[GridHub] Prognoza wzrostu OZE o %.1f%% — wyłączam elektrownię węglową\n",
				forecast.OZEChangePct)
			gh.conventional.Stop()
		}
	}
}

func (gh *GridHub) performBalance() {
	gh.mu.Lock()
	defer gh.mu.Unlock()

	gh.clearCurtailmentLocked()

	ozePower := 0.0
	for _, source := range gh.ozeSources {
		ozePower += source.GetCurrentPower()
	}

	if gh.conventional != nil {
		gh.conventional.Update()
	}
	convPower := 0.0
	if gh.conventional != nil {
		convPower = gh.conventional.GetCurrentPower()
	}

	totalDemand := 0.0
	for _, demand := range gh.currentDemands {
		totalDemand += demand
	}

	balance := (ozePower + convPower) - totalDemand
	status := "STABLE"

	if balance > 0 {
		if gh.ess != nil && gh.ess.GetSoC() < 1.0 {
			charged := gh.ess.Charge(balance)
			balance -= charged
			if charged > 0 {
				fmt.Printf("[GridHub] Ładowanie ESS: %.1f MWh (SoC: %.1f%%)\n",
					charged, gh.ess.GetSoC()*100)
			}
		}

		if balance > 0 && gh.ess != nil && gh.ess.GetSoC() >= 1.0 {
			if gh.conventional != nil && gh.conventional.GetStatus() == "Running" {
				fmt.Printf("[GridHub] Nadwyżka — wyłączam elektrownię węglową\n")
				gh.conventional.Stop()
				gh.conventional.Update()
				convPower = gh.conventional.GetCurrentPower()
				balance = (ozePower + convPower) - totalDemand
			}
		}

		if balance > 0 && len(gh.curtailableSources) > 0 {
			curtailed := gh.applyCurtailmentLocked(balance, ozePower)
			if curtailed > 0 {
				fmt.Printf("[GridHub] Curtailment: ograniczenie OZE o %.1f MW\n", curtailed)
				balance -= curtailed
			}
		}

		if balance > 0 {
			fmt.Printf("[GridHub] OSTRZEŻENIE: Nadal nadwyżka %.1f MW po ESS i Curtailment\n", balance)
			balance = 0
		}
	} else if balance < 0 {
		if gh.ess != nil && gh.ess.GetSoC() > 0 {
			dischargeNeeded := -balance
			discharged := gh.ess.Discharge(dischargeNeeded)
			balance += discharged
			if discharged > 0 {
				fmt.Printf("[GridHub] Rozładowanie ESS: %.1f MWh (SoC: %.1f%%)\n",
					discharged, gh.ess.GetSoC()*100)
			}
		}
	}

	if balance < -0.1 {
		status = "CRITICAL"
		gh.performLoadShedding(-balance)
	}

	if balance >= 0 && len(gh.sheddedConsumers) > 0 {
		fmt.Printf("[GridHub] Bilans poprawiony — przywracanie konsumentów\n")
		gh.sheddedConsumers = make(map[string]bool)
	}

	gh.dispatchSupplyStatusesLocked()

	soC := 0.0
	if gh.ess != nil {
		soC = gh.ess.GetSoC()
	}

	entry := LogEntry{
		GridStep:      gh.gridStep,
		WindSpeed:     0,
		SolarPct:      0,
		OZEPower:      ozePower,
		Conventional:  convPower,
		SoC:           soC,
		TotalDemand:   totalDemand,
		Balance:       balance,
		Status:        status,
		LoadShedCount: gh.loadShedCount,
		Timestamp:     time.Now(),
	}
	if gh.logger != nil {
		gh.logger.LogEntry(entry)
	}

	if gh.gridStep%ReportInterval == 0 {
		fmt.Printf("\n[=== RAPORT KROK %d ===]\n", gh.gridStep)
		fmt.Printf("[Pogoda] OZE: %.1f MW | Konwencjonalna: %.1f MW\n", ozePower, convPower)
		fmt.Printf("[Sieć] Popyt: %.1f MW | Bilans: %.1f MW | SoC: %.1f%% | Stan: %s\n",
			totalDemand, balance, soC*100, status)
		if gh.conventional != nil {
			fmt.Printf("[Elektrownia] Status: %s\n", gh.conventional.GetStatus())
		}
	}

	gh.currentDemands = make(map[string]float64)
}

func (gh *GridHub) clearCurtailmentLocked() {
	for _, source := range gh.curtailableSources {
		source.ClearCurtailment()
	}
}

func (gh *GridHub) applyCurtailmentLocked(surplus float64, ozePower float64) float64 {
	if surplus <= 0 || ozePower <= 0 {
		return 0
	}

	ratio := surplus / ozePower
	if ratio > 1.0 {
		ratio = 1.0
	}
	if ratio < 0 {
		ratio = 0
	}

	for _, source := range gh.curtailableSources {
		source.SetCurtailment(ratio)
	}

	return ozePower * ratio
}

func (gh *GridHub) dispatchSupplyStatusesLocked() {
	for _, consumer := range gh.consumers {
		id := consumer.GetID()
		demand := gh.currentDemands[id]
		status := SupplyStatus{
			ConsumerID:  id,
			AllocatedMW: demand,
			Reason:      "OK",
		}

		if gh.sheddedConsumers[id] {
			status.AllocatedMW = 0
			status.Reason = "LoadShed"
		}

		if ch, ok := gh.supplyChans[id]; ok {
			select {
			case ch <- status:
			default:
			}
		}
	}
}

func (gh *GridHub) performLoadShedding(deficit float64) {
	type consumerInfo struct {
		id       string
		priority int
		demand   float64
	}

	consumers := make([]consumerInfo, 0, len(gh.currentDemands))
	for id, demand := range gh.currentDemands {
		priority := 3
		for _, c := range gh.consumers {
			if c.GetID() == id {
				priority = c.GetPriority()
				break
			}
		}
		consumers = append(consumers, consumerInfo{id, priority, demand})
	}

	sort.Slice(consumers, func(i, j int) bool {
		return consumers[i].priority > consumers[j].priority
	})

	remainingDeficit := deficit
	for _, c := range consumers {
		if remainingDeficit <= 0 {
			break
		}
		if gh.sheddedConsumers[c.id] {
			continue
		}

		fmt.Printf("[GridHub] Load Shedding: odłączam %s (priorytet %d, zapotrzebowanie %.1f MW)\n",
			c.id, c.priority, c.demand)
		gh.sheddedConsumers[c.id] = true
		remainingDeficit -= c.demand

		gh.statsMu.Lock()
		gh.loadShedCount++
		gh.statsMu.Unlock()
	}

	if remainingDeficit > 0 {
		fmt.Printf("[GridHub] OSTRZEŻENIE: Nadal niedobór %.1f MW po Load Shedding!\n", remainingDeficit)
	}
}
