package main

import (
	"fmt"
	"sync"
)

type ConventionalPlant struct {
	name         string
	maxPower     float64
	status       string
	warmUpTime   int
	warmUpLeft   int
	currentPower float64
	mu           sync.RWMutex
}

func NewConventionalPlant(name string, maxPower float64, warmUpTime int) *ConventionalPlant {
	return &ConventionalPlant{
		name:       name,
		maxPower:   maxPower,
		status:     "Off",
		warmUpTime: warmUpTime,
		warmUpLeft: 0,
	}
}

func (cp *ConventionalPlant) Start() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if cp.status == "Off" || cp.status == "CoolingDown" {
		cp.status = "WarmingUp"
		cp.warmUpLeft = cp.warmUpTime
		fmt.Printf("[%s] Rozpoczęcie rozruchu (warm-up: %d kroków)\n", cp.name, cp.warmUpTime)
	}
}

func (cp *ConventionalPlant) Stop() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if cp.status == "Running" || cp.status == "WarmingUp" {
		cp.status = "CoolingDown"
		cp.currentPower = 0
		fmt.Printf("[%s] Rozpoczęcie wyłączania\n", cp.name)
	}
}

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

func (cp *ConventionalPlant) GetCurrentPower() float64 {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return cp.currentPower
}

func (cp *ConventionalPlant) GetName() string {
	return cp.name
}

func (cp *ConventionalPlant) GetStatus() string {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return cp.status
}
