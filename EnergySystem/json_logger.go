package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// JSONDataLogger zapisuje te same logi w formacie JSON.
type JSONDataLogger struct {
	logChan chan LogEntry
	file    *os.File
	writer  *bufio.Writer
	encoder *json.Encoder
	mu      sync.Mutex
}

// NewJSONDataLogger tworzy plik JSON z czytelnym formatowaniem.
func NewJSONDataLogger(filename string) (*JSONDataLogger, error) {
	if err := os.MkdirAll("logs", 0755); err != nil {
		return nil, err
	}

	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	writer := bufio.NewWriter(file)
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")

	return &JSONDataLogger{
		logChan: make(chan LogEntry, 100),
		file:    file,
		writer:  writer,
		encoder: encoder,
	}, nil
}

// LogEntry nie blokuje reszty symulacji, jeśli logger chwilowo nie nadąża.
func (dl *JSONDataLogger) LogEntry(entry LogEntry) {
	select {
	case dl.logChan <- entry:
	default:
	}
}

// Run zapisuje kolejne wpisy do pliku aż do zatrzymania systemu.
func (dl *JSONDataLogger) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("[DataLogger] Zamykanie i zapisywanie JSON...")
			dl.drainAndFlush()
			return
		case entry := <-dl.logChan:
			dl.writeEntry(entry)
		}
	}
}

func (dl *JSONDataLogger) writeEntry(entry LogEntry) {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	_ = dl.encoder.Encode(entry)
}

// drainAndFlush opróżnia kolejkę logów, a dopiero potem domyka plik.
func (dl *JSONDataLogger) drainAndFlush() {
	for {
		select {
		case entry := <-dl.logChan:
			dl.writeEntry(entry)
		default:
			dl.flush()
			return
		}
	}
}

// flush domyka bufor i plik na zakończenie pracy.
func (dl *JSONDataLogger) flush() {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	_ = dl.writer.Flush()
	_ = dl.file.Close()
	fmt.Println("[DataLogger] Dane zapisane do pliku JSON.")
}
