package main

import (
	"heimdall/internal/api"
	"heimdall/internal/auth"
	"heimdall/internal/config"
	"heimdall/internal/core"
	"heimdall/internal/services/dockerctl"
	"heimdall/internal/services/workerclient"
	"heimdall/internal/storage"
	"log/slog"
	"os"
	"strconv"
	"time"
)

func main() {
	cfg := config.Load()

	store, err := storage.New(cfg.DBPath)
	if err != nil {
		slog.Error("failed to open storage", "error", err)
		os.Exit(1)
	}
	defer func(store *storage.Store) {
		err := store.Close()
		if err != nil {
			slog.Error("failed to close storage", "error", err)
		}
	}(store)

	activityLog := core.NewActivityLog(store, cfg.EventBufferSize)

	baseHandler := slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{Level: slog.LevelInfo},
	)

	slog.SetDefault(slog.New(activityLog.Handler(baseHandler)))

	slog.Info("controller: storage opened", "path", cfg.DBPath)

	authStore, err := auth.Load(
		store,
		cfg.AuthUsername,
		cfg.AuthPassword,
	)
	if err != nil {
		slog.Error("failed to load auth", "error", err)
		os.Exit(1)
	}

	sessionTimeout := cfg.SessionTimeout

	if v, found, err := store.GetSetting("session_timeout_seconds"); err == nil && found {
		if secs, err := strconv.Atoi(v); err == nil {
			sessionTimeout = time.Duration(secs) * time.Second
		}
	}

	sessions := auth.NewSessionManager(sessionTimeout)

	ctl := dockerctl.New(cfg.ControllableContainers)

	worker := workerclient.New(
		cfg.WorkerInternalURL,
		cfg.InternalToken,
	)

	status := core.NewStatusTracker()

	bus := core.NewEventBus()

	core.StartDBEventBridge(
		store,
		bus,
		1*time.Second,
	)

	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			activityLog.Prune(cfg.ActivityRetention)
		}
	}()

	srv := api.New(
		bus,
		store,
		authStore,
		sessions,
		ctl,
		worker,
		activityLog,
		status,
		cfg.ControllerContainer,
	)

	slog.Info(
		"controller api starting",
		"addr",
		cfg.APIAddr,
	)

	if err := srv.Start(cfg.APIAddr); err != nil {
		slog.Error(
			"controller api failed",
			"error",
			err,
		)
		os.Exit(1)
	}
}
