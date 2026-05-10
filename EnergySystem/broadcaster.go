package main

import "sync"

type Broadcaster struct {
	mu          sync.RWMutex
	subscribers []chan WeatherData
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subscribers: make([]chan WeatherData, 0),
	}
}

func (b *Broadcaster) Subscribe() chan WeatherData {
	ch := make(chan WeatherData, 5)
	b.mu.Lock()
	b.subscribers = append(b.subscribers, ch)
	b.mu.Unlock()
	return ch
}

func (b *Broadcaster) Unsubscribe(ch chan WeatherData) {
	b.mu.Lock()
	for i, sub := range b.subscribers {
		if sub == ch {
			b.subscribers = append(b.subscribers[:i], b.subscribers[i+1:]...)
			close(ch)
			break
		}
	}
	b.mu.Unlock()
}

func (b *Broadcaster) Broadcast(data WeatherData) {
	b.mu.RLock()
	subs := make([]chan WeatherData, len(b.subscribers))
	copy(subs, b.subscribers)
	b.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- data:
		default:
			// Subskrybent zajęty — porzucamy paczkę
		}
	}
}
