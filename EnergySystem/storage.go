package main

import "sync"

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

func (bs *BatteryStorage) GetSoC() float64 {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	return bs.soc
}

func (bs *BatteryStorage) GetCapacity() float64 {
	return bs.capacity
}
