package core

import (
	"log/slog"
	"time"
)

type EventLister interface {
	EventsSince(since time.Time) ([]Event, error)
}

// DBEventBridge polls storage for new rows and republishes them on a local
// EventBus, giving the controller a live SSE feed without sharing an
// in-process channel with the worker (impossible — different process).
func StartDBEventBridge(store EventLister, bus *EventBus, interval time.Duration) {
	go func() {
		last := time.Now()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			events, err := store.EventsSince(last)
			if err != nil {
				slog.Warn("db event bridge poll failed", "error", err)
				continue
			}
			if len(events) == 0 {
				continue
			}
			for _, e := range events {
				bus.Publish(e)
				if e.Timestamp.After(last) {
					last = e.Timestamp
				}
			}
		}
	}()
}
