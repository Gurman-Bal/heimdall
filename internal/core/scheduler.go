package core

import (
	"log"
	"time"
)

type Scheduler struct {
	plugins  []Plugin
	interval time.Duration
	bus      *EventBus
}

func NewScheduler(bus *EventBus, interval time.Duration) *Scheduler {
	return &Scheduler{bus: bus, interval: interval}
}

func (s *Scheduler) Register(p Plugin) {
	s.plugins = append(s.plugins, p)
}

// Run starts every registered plugin, then polls all of them on a fixed
// interval, publishing whatever events they return. Stops when stop is closed.
func (s *Scheduler) Run(stop <-chan struct{}) {
	for _, p := range s.plugins {
		if err := p.Start(); err != nil {
			log.Printf("[scheduler] %s failed to start: %v", p.Name(), err)
		}
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			for _, p := range s.plugins {
				events, err := p.Poll()
				if err != nil {
					log.Printf("[scheduler] %s poll error: %v", p.Name(), err)
					continue
				}
				for _, e := range events {
					s.bus.Publish(e)
				}
			}
		}
	}
}
