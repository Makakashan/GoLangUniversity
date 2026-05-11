package main

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// TrendPredictor analizuje historię pogody i szacuje zmianę OZE.
type TrendPredictor struct {
	weatherCh  chan WeatherData
	forecastCh chan ForecastReport
	buffer     []WeatherData
	bufferMu   sync.Mutex
}

// NewTrendPredictor podpina się do broadcastera i tworzy kanał prognozy.
func NewTrendPredictor(broadcaster *Broadcaster) *TrendPredictor {
	return &TrendPredictor{
		weatherCh:  broadcaster.Subscribe(),
		forecastCh: make(chan ForecastReport, 1),
		buffer:     make([]WeatherData, 0, PredictorBufferSize),
	}
}

func (tp *TrendPredictor) GetForecastChan() <-chan ForecastReport {
	return tp.forecastCh
}

// Run zbiera pogodę do bufora i cyklicznie wystawia nową prognozę.
func (tp *TrendPredictor) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	gridTicker := time.NewTicker(GridStep)
	defer gridTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("[Predictor] Zamykanie...")
			return

		case weather := <-tp.weatherCh:
			tp.bufferMu.Lock()
			tp.buffer = append(tp.buffer, weather)
			if len(tp.buffer) > PredictorBufferSize {
				tp.buffer = tp.buffer[len(tp.buffer)-PredictorBufferSize:]
			}
			tp.bufferMu.Unlock()

		case <-gridTicker.C:
			tp.bufferMu.Lock()
			bufferCopy := make([]WeatherData, len(tp.buffer))
			copy(bufferCopy, tp.buffer)
			tp.bufferMu.Unlock()

			if len(bufferCopy) >= 2 {
				forecast := tp.calculateForecast(bufferCopy)
				select {
				case tp.forecastCh <- forecast:
				default:
					select {
					case <-tp.forecastCh:
					default:
					}
					select {
					case tp.forecastCh <- forecast:
					default:
					}
				}
			}
		}
	}
}

// calculateForecast porównuje dwie połowy historii, aby wykryć trend.
func (tp *TrendPredictor) calculateForecast(buffer []WeatherData) ForecastReport {
	half := len(buffer) / 2
	if half == 0 {
		half = 1
	}

	var avgWindFirst, avgSolarFirst float64
	for i := 0; i < half && i < len(buffer); i++ {
		avgWindFirst += buffer[i].WindSpeed
		avgSolarFirst += (100 - buffer[i].Solar)
	}
	avgWindFirst /= float64(half)
	avgSolarFirst /= float64(half)

	var avgWindSecond, avgSolarSecond float64
	for i := half; i < len(buffer); i++ {
		avgWindSecond += buffer[i].WindSpeed
		avgSolarSecond += (100 - buffer[i].Solar)
	}
	count := float64(len(buffer) - half)
	if count > 0 {
		avgWindSecond /= count
		avgSolarSecond /= count
	}

	windTrend := 0.0
	if avgWindFirst > 0 {
		windTrend = (avgWindSecond - avgWindFirst) / avgWindFirst * 100
	}
	solarTrend := 0.0
	if avgSolarFirst > 0 {
		solarTrend = (avgSolarSecond - avgSolarFirst) / avgSolarFirst * 100
	}

	combinedTrend := windTrend*0.6 + solarTrend*0.4

	return ForecastReport{
		Horizon:      ForecastHorizon,
		OZEChangePct: combinedTrend,
		Confidence:   math.Min(1.0, float64(len(buffer))/float64(PredictorBufferSize)),
		Timestamp:    time.Now(),
	}
}
