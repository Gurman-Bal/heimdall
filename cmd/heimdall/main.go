package main

import (
	"log/slog"
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
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

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
		slog.Error("failed to open storage", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	slog.Info("storage opened", "path", dbPath)

	existing, err := store.ListSources("truenas")
	if err != nil {
		slog.Error("failed to load sources", "error", err)
		os.Exit(1)
	}
	if len(existing) == 0 {
		defaults := []string{
			logDir + "/messages",
			logDir + "/auth.log",
			logDir + "/middlewared.log",
		}
		for _, p := range defaults {
			if _, err := store.AddSource("truenas", p); err != nil {
				slog.Error("failed to seed source", "path", p, "error", err)
			} else {
				slog.Info("seeded default source", "type", "truenas", "path", p)
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
				slog.Error("failed to save event", "error", err)
			}
			if e.Severity != "info" {
				slog.Warn("event detected", "severity", e.Severity, "source", e.Source, "type", e.Type, "message", e.Message)
			}
		}
	}()

	sources := map[string]api.ManagedSource{
		"truenas": truenasSource,
	}
	srv := api.New(bus, store, sources)
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

	slog.Info("heimdall watching", "path_count", len(paths), "paths", paths)
	<-sig
	slog.Info("shutting down")
	close(stop)
}
