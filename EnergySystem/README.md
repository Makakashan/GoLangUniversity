# System Współbieżnego Zarządzania Rozproszoną Siecią Energetyczną

Projekt реализует симулятор энергосети na Go z użyciem gorutyn, kanałów i graceful shutdown.

## Jak działa system

Struktura jest już dopasowana do diagramu:

1. **Warstwa wejściowa**
   - `WeatherStation` generuje dane pogodowe co `WeatherStep`.
   - `Broadcaster` rozsyła pogodę do subskrybentów.
   - `WindFarm` i `SolarFarm` przeliczają pogodę na produkcję OZE.
   - `TrendPredictor` analizuje historię i tworzy prognozę zmian OZE.

2. **Mózg systemu**
   - `GridHub` zbiera popyt, prognozę i produkcję.
   - Bilansuje moc.
   - Obsługuje `ESS`.
   - Uruchamia / zatrzymuje elektrownię węglową.
   - Robi `Curtailment` dla OZE, gdy jest nadwyżka.
   - Wysyła `SupplyStatus` do konsumentów.
   - Wykonuje `Load Shedding` wg priorytetów.

3. **Warstwa wykonawcza i buforowa**
   - `ConventionalPlant` ma rozruch (`WarmUp`) i wyłączanie.
   - `BatteryStorage` buforuje energię.

4. **Archiwizacja**
   - `CSVDataLogger` zapisuje do CSV.
   - `JSONDataLogger` zapisuje do JSON.

5. **Konsumenci**
   - `ResidentialConsumer`
   - `IndustrialConsumer`
   - `CriticalConsumer`

## Struktura plików

- `main.go` — tylko wiring i uruchamianie komponentów
- `types.go` — interfejsy, stałe i struktury danych
- `broadcaster.go` — pub/sub dla danych pogodowych
- `weather.go` — `WeatherStation`, `WindFarm`, `SolarFarm`
- `predictor.go` — `TrendPredictor`
- `conventional.go` — elektrownia konwencjonalna
- `storage.go` — magazyn energii / bateria
- `consumers.go` — konsumenci
- `gridhub.go` — centralne bilansowanie sieci
- `logger.go` — logger CSV
- `json_logger.go` — logger JSON

## Skalowanie czasu

- `WeatherStep`: 5 ms
- `GridStep`: 60 ms

## Uruchomienie

```bash
go build -o grid_simulator .
./grid_simulator
```

Jeśli chcesz JSON zamiast CSV:

```bash
LOGGER_FORMAT=json ./grid_simulator
```

Zatrzymanie symulacji: `Ctrl+C`

## Co już działa zgodnie z diagramem

- dane pogodowe idą przez `Broadcaster`
- OZE i predictor są podpięte do warstwy wejściowej
- `GridHub` steruje `ESS`, elektrownią węglową i curtailmentem
- konsumenci dostają `SupplyStatus`
- logowanie działa asynchronicznie
- dynamiczna rejestracja nowego konsumenta nadal działa
