package main

import (
	"context"
	"sync"
	"time"
)

const (
	WeatherStep         = 5 * time.Millisecond        // ~5 minut czasu symulacji
	GridStep            = 100 * time.Millisecond      // 1 godzina czasu symulacji
	WeatherPerGrid      = int(GridStep / WeatherStep) // 12 kroków pogodowych
	ForecastHorizon     = 5                           // prognoza na 5 kroków GridStep
	PredictorBufferSize = WeatherPerGrid * 2          // bufor = 2 godziny historii
	ReportInterval      = 5                           // raport co 5 kroków GridStep
)

type EnergySource interface {
	GetCurrentPower() float64
	GetName() string
}

type CurtailableSource interface {
	EnergySource
	SetCurtailment(ratio float64)
	ClearCurtailment()
}

type Predictor interface {
	GetForecastChan() <-chan ForecastReport
	Run(ctx context.Context, wg *sync.WaitGroup)
}

type Consumer interface {
	GetID() string
	GetPriority() int
	Run(ctx context.Context, wg *sync.WaitGroup, demandChan chan<- DemandReport, supplyChan <-chan SupplyStatus)
}

type EnergyStorage interface {
	Charge(amount float64) float64
	Discharge(amount float64) float64
	GetSoC() float64
	GetCapacity() float64
}

type WeatherProvider interface {
	Run(ctx context.Context, wg *sync.WaitGroup, broadcaster *Broadcaster)
}

type DataLogger interface {
	LogEntry(entry LogEntry)
	Run(ctx context.Context, wg *sync.WaitGroup)
}

type WeatherData struct {
	WindSpeed float64
	Solar     float64
	Timestamp time.Time
}

type DemandReport struct {
	ID        string
	Priority  int
	DemandMW  float64
	Timestamp time.Time
}

type SupplyStatus struct {
	ConsumerID  string
	AllocatedMW float64
	Reason      string // "OK", "LoadShed", "Partial"
}

type ForecastReport struct {
	Horizon      int
	OZEChangePct float64 // zmiana produkcji OZE w %
	Confidence   float64
	Timestamp    time.Time
}

type LogEntry struct {
	GridStep      int
	WindSpeed     float64
	SolarPct      float64
	OZEPower      float64
	Conventional  float64
	SoC           float64
	TotalDemand   float64
	Balance       float64
	Status        string
	LoadShedCount int
	Timestamp     time.Time
}
