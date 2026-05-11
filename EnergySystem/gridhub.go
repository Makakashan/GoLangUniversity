package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	conventionalMinUpSteps   = 3
	conventionalMinDownSteps = 3
)

// GridHub jest centralnym dispatcherem: zbiera dane, bilansuje sieć i wysyła decyzje.
type GridHub struct {
	ozeSources         []EnergySource
	curtailableSources []CurtailableSource
	conventional       *ConventionalPlant
	ess                EnergyStorage
	consumers          []Consumer
	supplyChans        map[string]chan SupplyStatus
	demandChan         chan DemandReport
	weatherChan        <-chan WeatherData
	forecastChan       <-chan ForecastReport
	logger             DataLogger

	currentDemands   map[string]float64
	sheddedConsumers map[string]bool
	lastWeather      WeatherData
	gridStep         int

	mu            sync.RWMutex
	statsMu       sync.Mutex
	loadShedCount int

	conventionalStartedStep  int
	conventionalLastStopStep int
}

// NewGridHub tworzy główny węzeł sterujący symulacji.
func NewGridHub(weatherChan <-chan WeatherData, forecastChan <-chan ForecastReport, logger DataLogger) *GridHub {
	return &GridHub{
		ozeSources:               make([]EnergySource, 0),
		curtailableSources:       make([]CurtailableSource, 0),
		consumers:                make([]Consumer, 0),
		supplyChans:              make(map[string]chan SupplyStatus),
		demandChan:               make(chan DemandReport, 100),
		weatherChan:              weatherChan,
		forecastChan:             forecastChan,
		logger:                   logger,
		currentDemands:           make(map[string]float64),
		sheddedConsumers:         make(map[string]bool),
		gridStep:                 0,
		loadShedCount:            0,
		conventionalStartedStep:  -1,
		conventionalLastStopStep: -conventionalMinDownSteps,
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

// Run odbiera popyt, pogodę i prognozy, a następnie uruchamia bilansowanie.
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

		case weather := <-gh.weatherChan:
			gh.mu.Lock()
			gh.lastWeather = weather
			gh.mu.Unlock()

		case forecast := <-gh.forecastChan:
			gh.handleForecast(forecast)

		case <-ticker.C:
			gh.performBalance()
			gh.gridStep++
		}
	}
}

// handleForecast uruchamia lub zatrzymuje elektrownię węglową z wyprzedzeniem.
func (gh *GridHub) handleForecast(forecast ForecastReport) {
	if forecast.OZEChangePct < -10 && forecast.Confidence > 0.5 {
		if gh.conventional != nil && gh.conventional.GetStatus() == "Off" &&
			(gh.conventionalLastStopStep < 0 || gh.gridStep-gh.conventionalLastStopStep >= conventionalMinDownSteps) {
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

// performBalance wykonuje jeden pełny krok decyzji energetycznych.
func (gh *GridHub) performBalance() {
	gh.mu.Lock()
	defer gh.mu.Unlock()

	gh.clearCurtailmentLocked()

	ozePower := 0.0
	for _, source := range gh.ozeSources {
		ozePower += source.GetCurrentPower()
	}

	prevConvStatus := ""
	if gh.conventional != nil {
		prevConvStatus = gh.conventional.GetStatus()
		gh.conventional.Update()
	}
	convPower := 0.0
	if gh.conventional != nil {
		convPower = gh.conventional.GetCurrentPower()
		if prevConvStatus != "Running" && gh.conventional.GetStatus() == "Running" {
			gh.conventionalStartedStep = gh.gridStep
		}
	}

	totalDemand := 0.0
	for _, demand := range gh.currentDemands {
		totalDemand += demand
	}

	balance := (ozePower + convPower) - totalDemand
	status := "STABLE"
	stepEvents := make([]string, 0, 4)

	if balance > 0 {
		if gh.ess != nil && gh.ess.GetSoC() < 1.0 {
			charged := gh.ess.Charge(balance)
			balance -= charged
			if charged > 0 {
				stepEvents = append(stepEvents, fmt.Sprintf("ESS charge %.1f MWh (SoC %.1f%%)", charged, gh.ess.GetSoC()*100))
			}
		}

		if balance > 0 && gh.ess != nil && gh.ess.GetSoC() >= 1.0 {
			if gh.conventional != nil && gh.conventional.GetStatus() == "Running" &&
				gh.conventionalStartedStep >= 0 &&
				gh.gridStep-gh.conventionalStartedStep >= conventionalMinUpSteps &&
				(gh.gridStep-gh.conventionalLastStopStep >= conventionalMinDownSteps) {
				gh.conventional.Stop()
				gh.conventional.Update()
				convPower = gh.conventional.GetCurrentPower()
				balance = (ozePower + convPower) - totalDemand
				gh.conventionalLastStopStep = gh.gridStep
				gh.conventionalStartedStep = -1
				stepEvents = append(stepEvents, "Coal plant stopped")
			}
		}

		if balance > 0 && len(gh.curtailableSources) > 0 {
			curtailed := gh.applyCurtailmentLocked(balance, ozePower)
			if curtailed > 0 {
				stepEvents = append(stepEvents, fmt.Sprintf("Curtailment %.1f MW", curtailed))
				balance -= curtailed
			}
		}

		if balance > 0 {
			stepEvents = append(stepEvents, fmt.Sprintf("Residual surplus %.1f MW", balance))
			balance = 0
		}
	} else if balance < 0 {
		if gh.ess != nil && gh.ess.GetSoC() > 0 {
			dischargeNeeded := -balance
			discharged := gh.ess.Discharge(dischargeNeeded)
			balance += discharged
			if discharged > 0 {
				stepEvents = append(stepEvents, fmt.Sprintf("ESS discharge %.1f MWh (SoC %.1f%%)", discharged, gh.ess.GetSoC()*100))
			}
		}
	}

	if balance < -0.1 {
		status = "CRITICAL"
		for _, msg := range gh.performLoadShedding(-balance) {
			stepEvents = append(stepEvents, msg)
		}
	}

	if balance >= 0 && len(gh.sheddedConsumers) > 0 {
		gh.sheddedConsumers = make(map[string]bool)
		stepEvents = append(stepEvents, "Consumers restored")
	}

	if len(stepEvents) > 0 {
		fmt.Printf("[GridHub] Step %d: %s\n", gh.gridStep, strings.Join(stepEvents, " | "))
	}

	gh.dispatchSupplyStatusesLocked()

	soC := 0.0
	windSpeed := 0.0
	solarStrength := 0.0
	if gh.ess != nil {
		soC = gh.ess.GetSoC()
	}
	windSpeed = gh.lastWeather.WindSpeed
	solarStrength = gh.lastWeather.Solar

	entry := LogEntry{
		GridStep:      gh.gridStep,
		WindSpeed:     windSpeed,
		SolarStrength: solarStrength,
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
		fmt.Printf("[Pogoda] Wiatr: %.1f m/s | Słońce: %.1f/100\n", windSpeed, solarStrength)
		fmt.Printf("[Generacja] OZE: %.1f MW | Konwencjonalna: %.1f MW\n", ozePower, convPower)
		fmt.Printf("[Sieć] Popyt: %.1f MW | Bilans: %.1f MW | SoC: %.1f%% | Stan: %s\n",
			totalDemand, balance, soC*100, status)
		if gh.conventional != nil {
			fmt.Printf("[Elektrownia] Status: %s\n", gh.conventional.GetStatus())
		}
	}

	gh.currentDemands = make(map[string]float64)
}

// clearCurtailmentLocked resetuje ograniczenia OZE przed kolejnym krokiem.
func (gh *GridHub) clearCurtailmentLocked() {
	for _, source := range gh.curtailableSources {
		source.ClearCurtailment()
	}
}

// applyCurtailmentLocked obcina OZE proporcjonalnie do nadwyżki energii.
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

// dispatchSupplyStatusesLocked wysyła każdemu konsumentowi bieżący stan zasilania.
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

// performLoadShedding odłącza odbiorców w kolejności od najmniej istotnych.
func (gh *GridHub) performLoadShedding(deficit float64) []string {
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

	events := make([]string, 0, len(consumers)+1)
	remainingDeficit := deficit
	for _, c := range consumers {
		if remainingDeficit <= 0 {
			break
		}
		if gh.sheddedConsumers[c.id] {
			continue
		}

		gh.sheddedConsumers[c.id] = true
		remainingDeficit -= c.demand
		events = append(events, fmt.Sprintf("shed %s (priorytet %d, %.1f MW)", c.id, c.priority, c.demand))

		gh.statsMu.Lock()
		gh.loadShedCount++
		gh.statsMu.Unlock()
	}

	if remainingDeficit > 0 {
		events = append(events, fmt.Sprintf("remaining deficit %.1f MW", remainingDeficit))
	}

	return events
}
