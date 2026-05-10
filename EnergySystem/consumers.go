package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type BaseConsumer struct {
	id         string
	priority   int
	demandChan chan<- DemandReport
	supplyChan <-chan SupplyStatus
}

type ResidentialConsumer struct {
	BaseConsumer
	baseDemand float64
}

func NewResidentialConsumer(id string, baseDemand float64) *ResidentialConsumer {
	return &ResidentialConsumer{
		BaseConsumer: BaseConsumer{
			id:       id,
			priority: 3,
		},
		baseDemand: baseDemand,
	}
}

func (rc *ResidentialConsumer) GetID() string    { return rc.id }
func (rc *ResidentialConsumer) GetPriority() int { return rc.priority }

func (rc *ResidentialConsumer) Run(ctx context.Context, wg *sync.WaitGroup, demandChan chan<- DemandReport, supplyChan <-chan SupplyStatus) {
	defer wg.Done()

	rc.demandChan = demandChan
	rc.supplyChan = supplyChan

	ticker := time.NewTicker(GridStep)
	defer ticker.Stop()

	gridStep := 0
	enabled := true
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("[%s] Zamykanie...\n", rc.id)
			return
		case status := <-rc.supplyChan:
			switch status.Reason {
			case "LoadShed":
				enabled = false
				fmt.Printf("[%s] Otrzymano odłączenie z sieci\n", rc.id)
			case "OK", "Partial":
				enabled = true
			}
		case <-ticker.C:
			hour := gridStep % 24
			demand := 0.0
			if enabled {
				demand = rc.baseDemand

				if hour >= 7 && hour <= 9 {
					demand *= 1.8
				}
				if hour >= 18 && hour <= 22 {
					demand *= 2.0
				}
				if hour >= 23 || hour <= 5 {
					demand *= 0.4
				}

				demand *= (0.9 + rand.Float64()*0.2)
			}

			report := DemandReport{
				ID:        rc.id,
				Priority:  rc.priority,
				DemandMW:  demand,
				Timestamp: time.Now(),
			}

			select {
			case rc.demandChan <- report:
			default:
			}

			gridStep++
		}
	}
}

type IndustrialConsumer struct {
	BaseConsumer
	baseDemand float64
}

func NewIndustrialConsumer(id string, baseDemand float64) *IndustrialConsumer {
	return &IndustrialConsumer{
		BaseConsumer: BaseConsumer{
			id:       id,
			priority: 2,
		},
		baseDemand: baseDemand,
	}
}

func (ic *IndustrialConsumer) GetID() string    { return ic.id }
func (ic *IndustrialConsumer) GetPriority() int { return ic.priority }

func (ic *IndustrialConsumer) Run(ctx context.Context, wg *sync.WaitGroup, demandChan chan<- DemandReport, supplyChan <-chan SupplyStatus) {
	defer wg.Done()

	ic.demandChan = demandChan
	ic.supplyChan = supplyChan

	ticker := time.NewTicker(GridStep)
	defer ticker.Stop()

	gridStep := 0
	enabled := true
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("[%s] Zamykanie...\n", ic.id)
			return
		case status := <-ic.supplyChan:
			switch status.Reason {
			case "LoadShed":
				enabled = false
				fmt.Printf("[%s] Otrzymano odłączenie z sieci\n", ic.id)
			case "OK", "Partial":
				enabled = true
			}
		case <-ticker.C:
			hour := gridStep % 24
			demand := 0.0
			if enabled {
				demand = ic.baseDemand

				if hour >= 6 && hour <= 18 {
					if rand.Float64() < 0.15 {
						demand *= 1.5
						fmt.Printf("[%s] Nagły pik zapotrzebowania!\n", ic.id)
					}
				} else {
					demand *= 0.15
				}
			}

			report := DemandReport{
				ID:        ic.id,
				Priority:  ic.priority,
				DemandMW:  demand,
				Timestamp: time.Now(),
			}

			select {
			case ic.demandChan <- report:
			default:
			}

			gridStep++
		}
	}
}

type CriticalConsumer struct {
	BaseConsumer
	baseDemand float64
}

func NewCriticalConsumer(id string, baseDemand float64) *CriticalConsumer {
	return &CriticalConsumer{
		BaseConsumer: BaseConsumer{
			id:       id,
			priority: 1,
		},
		baseDemand: baseDemand,
	}
}

func (cc *CriticalConsumer) GetID() string    { return cc.id }
func (cc *CriticalConsumer) GetPriority() int { return cc.priority }

func (cc *CriticalConsumer) Run(ctx context.Context, wg *sync.WaitGroup, demandChan chan<- DemandReport, supplyChan <-chan SupplyStatus) {
	defer wg.Done()

	cc.demandChan = demandChan
	cc.supplyChan = supplyChan

	ticker := time.NewTicker(GridStep)
	defer ticker.Stop()

	enabled := true
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("[%s] Zamykanie...\n", cc.id)
			return
		case status := <-cc.supplyChan:
			switch status.Reason {
			case "LoadShed":
				enabled = false
				fmt.Printf("[%s] Otrzymano odłączenie z sieci\n", cc.id)
			case "OK", "Partial":
				enabled = true
			}
		case <-ticker.C:
			demand := 0.0
			if enabled {
				demand = cc.baseDemand * (0.95 + rand.Float64()*0.1)
			}

			report := DemandReport{
				ID:        cc.id,
				Priority:  cc.priority,
				DemandMW:  demand,
				Timestamp: time.Now(),
			}

			select {
			case cc.demandChan <- report:
			default:
			}
		}
	}
}
