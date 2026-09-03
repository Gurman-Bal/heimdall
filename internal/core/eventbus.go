package core

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

type EventBus struct {
	mu          sync.RWMutex
	subscribers []chan Event
	dropped     atomic.Int64
}

func NewEventBus() *EventBus {
	b := &EventBus{}
	go b.reportDrops()
	return b
}

func (b *EventBus) Subscribe(buffer int) <-chan Event {
	ch := make(chan Event, buffer)
	b.mu.Lock()
	b.subscribers = append(b.subscribers, ch)
	b.mu.Unlock()
	return ch
}

func (b *EventBus) Publish(e Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subscribers {
		select {
		case ch <- e:
		default:
			b.dropped.Add(1)
		}
	}
}

func (b *EventBus) DroppedCount() int64 {
	return b.dropped.Load()
}

// reportDrops summarizes drops periodically rather than logging one line per
// dropped event — a per-event log during exactly the burst that causes drops
// would itself add to the overload.
func (b *EventBus) reportDrops() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	var last int64
	for range ticker.C {
		current := b.dropped.Load()
		if current > last {
			slog.Warn("event bus dropped events — buffer full, increase HEIMDALL_EVENT_BUFFER_SIZE if this recurs",
				"dropped_since_last_report", current-last, "total_dropped", current)
		}
		last = current
	}
}
