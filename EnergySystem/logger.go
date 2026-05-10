package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"sync"
	"time"
)

type CSVDataLogger struct {
	logChan   chan LogEntry
	file      *os.File
	writer    *bufio.Writer
	csvWriter *csv.Writer
	mu        sync.Mutex
}

func NewCSVDataLogger(filename string) (*CSVDataLogger, error) {
	os.MkdirAll("logs", 0755)
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	writer := bufio.NewWriter(file)
	csvWriter := csv.NewWriter(writer)

	csvWriter.Write([]string{
		"GridStep", "WindSpeed", "SolarPct", "OZEPower", "Conventional",
		"SoC", "TotalDemand", "Balance", "Status", "LoadShedCount", "Timestamp",
	})
	csvWriter.Flush()

	return &CSVDataLogger{
		logChan:   make(chan LogEntry, 100),
		file:      file,
		writer:    writer,
		csvWriter: csvWriter,
	}, nil
}

func (dl *CSVDataLogger) LogEntry(entry LogEntry) {
	select {
	case dl.logChan <- entry:
	default:
	}
}

func (dl *CSVDataLogger) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("[DataLogger] Zamykanie i zapisywanie...")
			dl.flush()
			return
		case entry := <-dl.logChan:
			dl.writeEntry(entry)
		}
	}
}

func (dl *CSVDataLogger) writeEntry(entry LogEntry) {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	record := []string{
		fmt.Sprintf("%d", entry.GridStep),
		fmt.Sprintf("%.2f", entry.WindSpeed),
		fmt.Sprintf("%.2f", entry.SolarPct),
		fmt.Sprintf("%.2f", entry.OZEPower),
		fmt.Sprintf("%.2f", entry.Conventional),
		fmt.Sprintf("%.4f", entry.SoC),
		fmt.Sprintf("%.2f", entry.TotalDemand),
		fmt.Sprintf("%.2f", entry.Balance),
		entry.Status,
		fmt.Sprintf("%d", entry.LoadShedCount),
		entry.Timestamp.Format(time.RFC3339),
	}
	dl.csvWriter.Write(record)
}

func (dl *CSVDataLogger) flush() {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	dl.csvWriter.Flush()
	dl.writer.Flush()
	dl.file.Close()
	fmt.Println("[DataLogger] Dane zapisane do pliku.")
}
