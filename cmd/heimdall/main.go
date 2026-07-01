package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"heimdall/internal/api"
	"heimdall/internal/core"
	"heimdall/internal/ingest"
	_ "heimdall/internal/plugins/minecraft" // blank import = registers itself via init()
	_ "heimdall/internal/plugins/truenas"
	"heimdall/internal/storage"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	dbPath := os.Getenv("HEIMDALL_DB_PATH")
	if dbPath == "" {
		dbPath = "./heimdall.db"
	}

	store, err := storage.New(dbPath)
	if err != nil {
		slog.Error("failed to open storage", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	slog.Info("storage opened", "path", dbPath)

	// seed truenas defaults on first run only — minecraft has no sensible
	// default path, it stays empty until added via the UI
	logDir := os.Getenv("HEIMDALL_LOG_DIR")
	if logDir == "" {
		logDir = "./testlogs"
	}
	if existing, _ := store.ListSources("truenas"); len(existing) == 0 {
		for _, p := range []string{logDir + "/messages", logDir + "/auth.log", logDir + "/middlewared.log"} {
			if _, err := store.AddSource("truenas", p); err != nil {
				slog.Error("failed to seed source", "path", p, "error", err)
			}
		}
	}

	bus := core.NewEventBus()
	scheduler := core.NewScheduler(bus, 5*time.Second)
	managed := map[string]api.ManagedSource{}

	for _, sourceType := range ingest.Registered() {
		cfgs, err := store.ListSources(sourceType)
		if err != nil {
			slog.Error("failed to load sources", "type", sourceType, "error", err)
			continue
		}
		var paths []string
		for _, c := range cfgs {
			paths = append(paths, c.Path)
		}

		src, ok := ingest.New(sourceType, paths, store)
		if !ok {
			continue
		}
		scheduler.Register(src)
		managed[sourceType] = src
		slog.Info("source type initialized", "type", sourceType, "path_count", len(paths))
	}

	persistCh := bus.Subscribe(100)
	go func() {
		for e := range persistCh {
			if err := store.SaveEvent(e); err != nil {
				slog.Error("failed to save event", "error", err)
			}
			if e.Severity != "info" {
				slog.Warn("event detected", "severity", e.Severity, "source", e.Source, "type", e.Type, "message", e.Message)
			}
		}
	}()

	srv := api.New(bus, store, managed)
	go func() {
		slog.Info("api server starting", "addr", ":8080")
		if err := srv.Start(":8080"); err != nil {
			slog.Error("api server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan struct{})
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go scheduler.Run(stop)

	slog.Info("heimdall started", "registered_types", ingest.Registered())
	<-sig
	slog.Info("shutting down")
	close(stop)
}
