package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// buildLogger wybiera format logów na podstawie zmiennej środowiskowej.
func buildLogger() (DataLogger, error) {
	format := strings.ToLower(os.Getenv("LOGGER_FORMAT"))
	switch format {
	case "json":
		return NewJSONDataLogger("logs/grid_history.json")
	default:
		return NewCSVDataLogger("logs/grid_history.csv")
	}
}

// main tylko składa komponenty razem; cała logika siedzi w osobnych modułach.
func main() {
	rand.Seed(time.Now().UnixNano())
	fmt.Println("=== SYSTEM WSPOŁBIEŻNEGO ZARZĄDZANIA ENERGETYCZNEGO ===")
	fmt.Println("Inicjalizacja komponentów...")

	// 1. Kontekst i graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n[SYSTEM] Otrzymano sygnał przerwania. Zamykanie...")
		cancel()
	}()

	var wg sync.WaitGroup

	// 2. Broadcaster
	broadcaster := NewBroadcaster()

	// 3. WeatherStation
	weatherStation := NewWeatherStation()
	wg.Add(1)
	go weatherStation.Run(ctx, &wg, broadcaster)

	// 4. Predictor
	predictor := NewTrendPredictor(broadcaster)
	wg.Add(1)
	go predictor.Run(ctx, &wg)

	// 5. Farma OZE
	windFarm := NewWindFarm("FarmaWiatrowa-1", 200.0, broadcaster)
	solarFarm := NewSolarFarm("FarmaSloneczna-1", 150.0, broadcaster)

	wg.Add(1)
	go windFarm.Run(ctx, &wg)
	wg.Add(1)
	go solarFarm.Run(ctx, &wg)

	// 6. Elektrownia węglowa
	coalPlant := NewConventionalPlant("ElektrowniaWeglowa", 300.0, 3)
	wg.Add(1)
	go coalPlant.Run(ctx, &wg)

	// 7. ESS
	battery := NewBatteryStorage("BateriaGlowna", 500.0, 100.0, 0.5)

	// 8. DataLogger
	logger, err := buildLogger()
	if err != nil {
		fmt.Printf("Błąd tworzenia loggera: %v\n", err)
		return
	}
	wg.Add(1)
	go logger.Run(ctx, &wg)

	// 9. GridHub
	weatherChanForHub := broadcaster.Subscribe()
	gridHub := NewGridHub(weatherChanForHub, predictor.GetForecastChan(), logger)
	gridHub.AddOZESource(windFarm)
	gridHub.AddOZESource(solarFarm)
	gridHub.SetConventional(coalPlant.GetCommandChan(), coalPlant.GetReportChan())
	gridHub.SetESS(battery)

	// 10. Konsumenci (Fan-In + SupplyStatus feedback)
	demandChan := gridHub.GetDemandChan()

	res1 := NewResidentialConsumer("Dom-1", 5.0)
	res2 := NewResidentialConsumer("Dom-2", 8.0)
	res3 := NewResidentialConsumer("Dom-3", 6.0)
	ind1 := NewIndustrialConsumer("Fabryka-1", 50.0)
	ind2 := NewIndustrialConsumer("Fabryka-2", 40.0)
	crit1 := NewCriticalConsumer("Szpital", 15.0)
	crit2 := NewCriticalConsumer("Policja", 5.0)

	consumers := []Consumer{res1, res2, res3, ind1, ind2, crit1, crit2}

	for _, consumer := range consumers {
		supplyChan := gridHub.RegisterConsumer(consumer)
		wg.Add(1)
		go consumer.Run(ctx, &wg, demandChan, supplyChan)
	}

	// 11. Uruchomienie GridHub
	wg.Add(1)
	go gridHub.Run(ctx, &wg)

	// 12. Dynamiczna rejestracja nowego konsumenta
	// Pokazuje, że sieć może zmieniać się w trakcie działania symulacji.
	go func() {
		time.Sleep(30 * GridStep)
		select {
		case <-ctx.Done():
			return
		default:
			newConsumer := NewIndustrialConsumer("NowaFabryka-3", 35.0)
			supplyChan := gridHub.RegisterConsumer(newConsumer)
			wg.Add(1)
			go newConsumer.Run(ctx, &wg, demandChan, supplyChan)
			fmt.Println("\n[SYSTEM] Dynamicznie dodano nowego konsumenta: NowaFabryka-3")
		}
	}()

	// 13. Czekamy na zakończenie
	wg.Wait()
	fmt.Println("\n[SYSTEM] Wszystkie komponenty zamknięte. Koniec symulacji.")
}
