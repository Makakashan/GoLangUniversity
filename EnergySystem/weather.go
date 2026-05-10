package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type WeatherStation struct {
	windSpeed float64
	solar     float64
}

func NewWeatherStation() *WeatherStation {
	return &WeatherStation{
		windSpeed: 15.0 + rand.Float64()*10,
		solar:     30.0 + rand.Float64()*20,
	}
}

func (ws *WeatherStation) Run(ctx context.Context, wg *sync.WaitGroup, broadcaster *Broadcaster) {
	defer wg.Done()
	ticker := time.NewTicker(WeatherStep)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("[WeatherStation] Zamykanie...")
			return
		case t := <-ticker.C:
			ws.windSpeed += rand.Float64()*2 - 1
			if ws.windSpeed < 0 {
				ws.windSpeed = 0
			}
			if ws.windSpeed > 50 {
				ws.windSpeed = 50
			}

			ws.solar += rand.Float64()*4 - 2
			if ws.solar < 0 {
				ws.solar = 0
			}
			if ws.solar > 100 {
				ws.solar = 100
			}

			data := WeatherData{
				WindSpeed: ws.windSpeed,
				Solar:     ws.solar,
				Timestamp: t,
			}
			broadcaster.Broadcast(data)
		}
	}
}

type WindFarm struct {
	name             string
	maxPower         float64
	weatherCh        chan WeatherData
	rawPower         float64
	curtailmentRatio float64
	mu               sync.RWMutex
}

func NewWindFarm(name string, maxPower float64, broadcaster *Broadcaster) *WindFarm {
	return &WindFarm{
		name:      name,
		maxPower:  maxPower,
		weatherCh: broadcaster.Subscribe(),
	}
}

func (wf *WindFarm) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("[%s] Zamykanie...\n", wf.name)
			return
		case weather := <-wf.weatherCh:
			power := 0.0
			if weather.WindSpeed > 3 {
				if weather.WindSpeed >= 25 {
					power = wf.maxPower
				} else {
					power = wf.maxPower * (weather.WindSpeed / 25.0)
				}
			}
			wf.mu.Lock()
			wf.rawPower = power
			wf.mu.Unlock()
		}
	}
}

func (wf *WindFarm) GetCurrentPower() float64 {
	wf.mu.RLock()
	defer wf.mu.RUnlock()
	return wf.rawPower * (1.0 - wf.curtailmentRatio)
}

func (wf *WindFarm) SetCurtailment(ratio float64) {
	wf.mu.Lock()
	defer wf.mu.Unlock()
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	wf.curtailmentRatio = ratio
}

func (wf *WindFarm) ClearCurtailment() {
	wf.SetCurtailment(0)
}

func (wf *WindFarm) GetName() string {
	return wf.name
}

type SolarFarm struct {
	name             string
	maxPower         float64
	weatherCh        chan WeatherData
	rawPower         float64
	curtailmentRatio float64
	mu               sync.RWMutex
}

func NewSolarFarm(name string, maxPower float64, broadcaster *Broadcaster) *SolarFarm {
	return &SolarFarm{
		name:      name,
		maxPower:  maxPower,
		weatherCh: broadcaster.Subscribe(),
	}
}

func (sf *SolarFarm) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("[%s] Zamykanie...\n", sf.name)
			return
		case weather := <-sf.weatherCh:
			sunPct := weather.Solar / 100.0
			power := sf.maxPower * sunPct
			sf.mu.Lock()
			sf.rawPower = power
			sf.mu.Unlock()
		}
	}
}

func (sf *SolarFarm) GetCurrentPower() float64 {
	sf.mu.RLock()
	defer sf.mu.RUnlock()
	return sf.rawPower * (1.0 - sf.curtailmentRatio)
}

func (sf *SolarFarm) SetCurtailment(ratio float64) {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	sf.curtailmentRatio = ratio
}

func (sf *SolarFarm) ClearCurtailment() {
	sf.SetCurtailment(0)
}

func (sf *SolarFarm) GetName() string {
	return sf.name
}
