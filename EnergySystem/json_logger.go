package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type JSONDataLogger struct {
	logChan chan LogEntry
	file    *os.File
	writer  *bufio.Writer
	encoder *json.Encoder
	mu      sync.Mutex
}

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

func (dl *JSONDataLogger) LogEntry(entry LogEntry) {
	select {
	case dl.logChan <- entry:
	default:
	}
}

func (dl *JSONDataLogger) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("[DataLogger] Zamykanie i zapisywanie JSON...")
			dl.flush()
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

func (dl *JSONDataLogger) flush() {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	_ = dl.writer.Flush()
	_ = dl.file.Close()
	fmt.Println("[DataLogger] Dane zapisane do pliku JSON.")
}
