package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ConventionalCommand steruje pracą elektrowni poprzez kanał.
type ConventionalCommand string

const (
	ConventionalCommandStart ConventionalCommand = "START"
	ConventionalCommandStop  ConventionalCommand = "STOP"
)

// ConventionalReport przekazuje GridHub aktualny stan elektrowni.
type ConventionalReport struct {
	Name         string
	Status       string
	CurrentPower float64
	Timestamp    time.Time
}

// ConventionalPlant modeluje elektrownię z rozruchem, pracą i wyłączaniem.
type ConventionalPlant struct {
	name         string
	maxPower     float64
	status       string
	warmUpTime   int
	warmUpLeft   int
	currentPower float64

	commandChan chan ConventionalCommand
	reportChan  chan ConventionalReport
	mu          sync.RWMutex
}

func NewConventionalPlant(name string, maxPower float64, warmUpTime int) *ConventionalPlant {
	return &ConventionalPlant{
		name:        name,
		maxPower:    maxPower,
		status:      "Off",
		warmUpTime:  warmUpTime,
		warmUpLeft:  0,
		commandChan: make(chan ConventionalCommand, 10),
		reportChan:  make(chan ConventionalReport, 10),
	}
}

func (cp *ConventionalPlant) GetCommandChan() chan<- ConventionalCommand {
	return cp.commandChan
}

func (cp *ConventionalPlant) GetReportChan() <-chan ConventionalReport {
	return cp.reportChan
}

// Run obsługuje komendy z GridHub i cyklicznie aktualizuje stan elektrowni.
func (cp *ConventionalPlant) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(GridStep)
	defer ticker.Stop()

	cp.publishStatus()

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("[%s] Zamykanie...\n", cp.name)
			return
		case command := <-cp.commandChan:
			switch command {
			case ConventionalCommandStart:
				cp.Start()
			case ConventionalCommandStop:
				cp.Stop()
			}
			cp.publishStatus()
		case <-ticker.C:
			cp.Update()
			cp.publishStatus()
		}
	}
}

// Start uruchamia elektrownię i rozpoczyna etap rozgrzewania.
func (cp *ConventionalPlant) Start() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if cp.status == "Off" || cp.status == "CoolingDown" {
		cp.status = "WarmingUp"
		cp.warmUpLeft = cp.warmUpTime
		fmt.Printf("[%s] Rozpoczęcie rozruchu (warm-up: %d kroków)\n", cp.name, cp.warmUpTime)
	}
}

// Stop rozpoczyna kontrolowane wygaszanie stacji.
func (cp *ConventionalPlant) Stop() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if cp.status == "Running" || cp.status == "WarmingUp" {
		cp.status = "CoolingDown"
		cp.currentPower = 0
		fmt.Printf("[%s] Rozpoczęcie wyłączania\n", cp.name)
	}
}

// Update przesuwa stan elektrowni o jeden krok symulacji.
func (cp *ConventionalPlant) Update() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	switch cp.status {
	case "WarmingUp":
		cp.warmUpLeft--
		if cp.warmUpLeft <= 0 {
			cp.status = "Running"
			cp.currentPower = cp.maxPower
			fmt.Printf("[%s] Pełna moc osiągnięta! (%.1f MW)\n", cp.name, cp.maxPower)
		} else {
			progress := 1.0 - float64(cp.warmUpLeft)/float64(cp.warmUpTime)
			cp.currentPower = cp.maxPower * progress * 0.5
		}
	case "CoolingDown":
		cp.status = "Off"
		cp.currentPower = 0
		fmt.Printf("[%s] Wyłączona\n", cp.name)
	}
}

// GetCurrentPower zwraca aktualnie dostępną moc stacji.
func (cp *ConventionalPlant) GetCurrentPower() float64 {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return cp.currentPower
}

func (cp *ConventionalPlant) GetName() string {
	return cp.name
}

// GetStatus zwraca bieżący stan pracy elektrowni.
func (cp *ConventionalPlant) GetStatus() string {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return cp.status
}

func (cp *ConventionalPlant) publishStatus() {
	report := ConventionalReport{
		Name:         cp.GetName(),
		Status:       cp.GetStatus(),
		CurrentPower: cp.GetCurrentPower(),
		Timestamp:    time.Now(),
	}

	select {
	case cp.reportChan <- report:
	default:
	}
}
