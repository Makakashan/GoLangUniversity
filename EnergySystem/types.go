package main

import (
	"context"
	"sync"
	"time"
)

// Skala czasu symulacji: pogoda zmienia się częściej niż bilans sieci.
const (
	WeatherStep         = 5 * time.Millisecond        // ~5 minut czasu symulacji
	GridStep            = 60 * time.Millisecond       // 1 godzina czasu symulacji
	WeatherPerGrid      = int(GridStep / WeatherStep) // 12 kroków pogodowych
	ForecastHorizon     = 5                           // prognoza na 5 kroków GridStep
	PredictorBufferSize = WeatherPerGrid * 2          // bufor = 2 godziny historii
	ReportInterval      = 5                           // raport co 5 kroków GridStep
)

// Każde źródło energii musi ujawnić aktualną moc i nazwę.
type EnergySource interface {
	GetCurrentPower() float64
	GetName() string
}

// Źródła OZE mogą dodatkowo ograniczać swoją produkcję.
type CurtailableSource interface {
	EnergySource
	SetCurtailment(ratio float64)
	ClearCurtailment()
}

// Predictor analizuje pogodę i wystawia prognozę dla GridHub.
type Predictor interface {
	GetForecastChan() <-chan ForecastReport
	Run(ctx context.Context, wg *sync.WaitGroup)
}

// Konsument zgłasza popyt i odbiera stan zasilania z sieci.
type Consumer interface {
	GetID() string
	GetPriority() int
	Run(ctx context.Context, wg *sync.WaitGroup, demandChan chan<- DemandReport, supplyChan <-chan SupplyStatus)
}

// Magazyn energii przechowuje nadwyżkę i oddaje energię przy deficycie.
type EnergyStorage interface {
	Charge(amount float64) float64
	Discharge(amount float64) float64
	GetSoC() float64
	GetCapacity() float64
}

// Dostawca pogody publikuje dane do broadcastera.
type WeatherProvider interface {
	Run(ctx context.Context, wg *sync.WaitGroup, broadcaster *Broadcaster)
}

// Logger zapisuje historię pracy całej sieci.
type DataLogger interface {
	LogEntry(entry LogEntry)
	Run(ctx context.Context, wg *sync.WaitGroup)
}

// Aktualny stan pogody przesyłany po systemie.
type WeatherData struct {
	WindSpeed float64
	Solar     float64
	Timestamp time.Time
}

// Raport popytu wysyłany przez konsumenta do GridHub.
type DemandReport struct {
	ID        string
	Priority  int
	DemandMW  float64
	Timestamp time.Time
}

// Odpowiedź sieci dla konkretnego konsumenta.
type SupplyStatus struct {
	ConsumerID  string
	AllocatedMW float64
	Reason      string // "OK", "LoadShed", "Partial"
}

// Prognoza informująca o spodziewanej zmianie generacji OZE.
type ForecastReport struct {
	Horizon      int
	OZEChangePct float64 // zmiana produkcji OZE w %
	Confidence   float64
	Timestamp    time.Time
}

// Zapis jednego kroku symulacji do CSV/JSON.
type LogEntry struct {
	GridStep      int
	WindSpeed     float64
	SolarStrength float64
	OZEPower      float64
	Conventional  float64
	SoC           float64
	TotalDemand   float64
	Balance       float64
	Status        string
	LoadShedCount int
	Timestamp     time.Time
}
