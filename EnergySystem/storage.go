package main

import "sync"

// BatteryStorage buforuje energię i pilnuje limitu mocy ładowania/rozładowania.
type BatteryStorage struct {
	name     string
	capacity float64
	soc      float64
	maxRate  float64
	mu       sync.Mutex
}

func NewBatteryStorage(name string, capacity, maxRate float64, initialSoC float64) *BatteryStorage {
	return &BatteryStorage{
		name:     name,
		capacity: capacity,
		soc:      initialSoC,
		maxRate:  maxRate,
	}
}

// Charge przyjmuje nadwyżkę, ale nie przekracza pojemności ani maxRate.
func (bs *BatteryStorage) Charge(amount float64) float64 {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if bs.soc >= 1.0 {
		return 0
	}

	if amount > bs.maxRate {
		amount = bs.maxRate
	}

	available := bs.capacity * (1.0 - bs.soc)
	if amount > available {
		amount = available
	}

	bs.soc += amount / bs.capacity
	if bs.soc > 1.0 {
		bs.soc = 1.0
	}

	return amount
}

// Discharge oddaje energię do sieci w granicach dostępnego SoC i maxRate.
func (bs *BatteryStorage) Discharge(amount float64) float64 {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if bs.soc <= 0.0 {
		return 0
	}

	if amount > bs.maxRate {
		amount = bs.maxRate
	}

	available := bs.capacity * bs.soc
	if amount > available {
		amount = available
	}

	bs.soc -= amount / bs.capacity
	if bs.soc < 0.0 {
		bs.soc = 0.0
	}

	return amount
}

// GetSoC zwraca bieżący poziom naładowania w skali 0..1.
func (bs *BatteryStorage) GetSoC() float64 {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	return bs.soc
}

// GetCapacity zwraca maksymalną pojemność magazynu.
func (bs *BatteryStorage) GetCapacity() float64 {
	return bs.capacity
}
