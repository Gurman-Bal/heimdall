package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"heimdall/internal/api"
	"heimdall/internal/core"
	"heimdall/internal/plugins/truenas"
	"heimdall/internal/storage"
)

func main() {
	logDir := os.Getenv("HEIMDALL_LOG_DIR")
	if logDir == "" {
		logDir = "./testlogs"
	}
	dbPath := os.Getenv("HEIMDALL_DB_PATH")
	if dbPath == "" {
		dbPath = "./heimdall.db"
	}

	store, err := storage.New(dbPath)
	if err != nil {
		log.Fatalf("failed to open storage: %v", err)
	}
	defer store.Close()

	// seed default sources on first run only
	existing, err := store.ListSources("truenas")
	if err != nil {
		log.Fatalf("failed to load sources: %v", err)
	}
	if len(existing) == 0 {
		defaults := []string{
			logDir + "/messages",
			logDir + "/auth.log",
			logDir + "/middlewared.log",
		}
		for _, p := range defaults {
			if _, err := store.AddSource("truenas", p); err != nil {
				log.Printf("failed to seed source %s: %v", p, err)
			}
		}
		existing, _ = store.ListSources("truenas")
	}

	var paths []string
	for _, cfg := range existing {
		paths = append(paths, cfg.Path)
	}

	bus := core.NewEventBus()
	scheduler := core.NewScheduler(bus, 5*time.Second)

	truenasSource := truenas.New(paths, store)
	scheduler.Register(truenasSource)

	persistCh := bus.Subscribe(100)
	go func() {
		for e := range persistCh {
			if err := store.SaveEvent(e); err != nil {
				log.Printf("failed to save event: %v", err)
			}
			if e.Severity != "info" {
				log.Printf("[%s] %s/%s: %s", e.Severity, e.Source, e.Type, e.Message)
			}
		}
	}()

	sources := map[string]api.ManagedSource{
		"truenas": truenasSource,
	}
	srv := api.New(bus, store, sources)
	go func() {
		log.Println("heimdall api listening on :8080")
		if err := srv.Start(":8080"); err != nil {
			log.Fatalf("api server failed: %v", err)
		}
	}()

	stop := make(chan struct{})
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go scheduler.Run(stop)

	log.Printf("heimdall watching: %v", paths)
	<-sig
	log.Println("shutting down...")
	close(stop)
}
