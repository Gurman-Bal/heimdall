package main

import (
	"context"
	"heimdall/internal/api"
	"heimdall/internal/auth"
	"heimdall/internal/config"
	"heimdall/internal/core"
	"heimdall/internal/ingest"
	_ "heimdall/internal/plugins/minecraft"
	_ "heimdall/internal/plugins/truenas"
	"heimdall/internal/services/dockerctl"
	"heimdall/internal/services/reporting"
	"heimdall/internal/storage"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

type defaultRule struct {
	pattern   string
	severity  string
	eventType string
}

var defaultRules = map[string][]defaultRule{
	"truenas": {
		{`(?i)\b(reallocated sector|pending sector|smart.*fail)\b`, "critical", "smart_warning"},
		{`(?i)\b(panic|critical|failed|failure)\b`, "critical", "error"},
		{`(?i)\b(degraded|warn|warning)\b`, "warning", "warning"},
		{`(?i)\b(denied|refused|error)\b`, "warning", "error"},
	},
	"minecraft": {
		{`(?i)\bOutOfMemoryError\b`, "critical", "crash"},
		{`(?i)(Exception in server tick loop|server thread/FATAL)`, "critical", "crash"},
		{`(?i)Can't keep up! Is the server overloaded`, "warning", "tps_warning"},
		{`(?i)Lithium Class Analysis Error`, "info", "lithium_noise"},
		{`(?i)\b(ERROR|Exception)\b`, "warning", "error"},
		{`(?i)joined the game`, "info", "player_join"},
		{`(?i)left the game`, "info", "player_leave"},
	},
}

func seedDefaultRules(store *storage.Store, sourceType string) {
	existing, err := store.ListRules(sourceType)
	if err != nil {
		slog.Error("failed to check existing rules", "type", sourceType, "error", err)
		return
	}
	if len(existing) > 0 {
		return
	}
	for i, r := range defaultRules[sourceType] {
		if _, err := store.AddRule(sourceType, r.pattern, r.severity, r.eventType, (i+1)*10); err != nil {
			slog.Error("failed to seed rule", "type", sourceType, "error", err)
		}
	}
	slog.Info("seeded default rules", "type", sourceType, "count", len(defaultRules[sourceType]))
}

func loadRules(store *storage.Store, engine *core.RuleEngine, sourceType string) {
	cfgs, err := store.ListRules(sourceType)
	if err != nil {
		slog.Error("failed to load rules", "type", sourceType, "error", err)
		return
	}
	defs := make([]core.RuleDef, len(cfgs))
	for i, c := range cfgs {
		defs[i] = core.RuleDef{ID: c.ID, Pattern: c.Pattern, Severity: c.Severity, EventType: c.EventType, Priority: c.Priority}
	}
	if errs := engine.Load(sourceType, defs); len(errs) > 0 {
		for _, e := range errs {
			slog.Error("rule failed to compile", "type", sourceType, "error", e)
		}
	}
}

func main() {
	cfg := config.Load()

	store, err := storage.New(cfg.DBPath)
	if err != nil {
		slog.Error("failed to open storage", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	activityLog := core.NewActivityLog(store, cfg.EventBufferSize)
	baseHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(activityLog.Handler(baseHandler)))

	slog.Info("config loaded",
		"db_path", cfg.DBPath, "log_dir", cfg.DefaultLogDir, "api_addr", cfg.APIAddr,
		"ollama_url", cfg.OllamaURL, "llm_model", cfg.LLMModel, "report_interval", cfg.ReportInterval,
		"event_buffer_size", cfg.EventBufferSize, "batch_size", cfg.BatchSize,
		"session_timeout", cfg.SessionTimeout, "activity_retention", cfg.ActivityRetention)
	slog.Info("storage opened", "path", cfg.DBPath)

	authStore, err := auth.Load(store, cfg.AuthUsername, cfg.AuthPassword)
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
	status := core.NewStatusTracker()

	if existing, _ := store.ListSources("truenas"); len(existing) == 0 {
		for _, p := range []string{cfg.DefaultLogDir + "/messages", cfg.DefaultLogDir + "/auth.log", cfg.DefaultLogDir + "/middlewared.log"} {
			if _, err := store.AddSource("truenas", p); err != nil {
				slog.Error("failed to seed source", "path", p, "error", err)
			}
		}
	}

	ruleEngine := core.NewRuleEngine()
	for sourceType := range defaultRules {
		seedDefaultRules(store, sourceType)
	}

	bus := core.NewEventBus()
	scheduler := core.NewScheduler(bus, 5*time.Second)
	managed := map[string]api.ManagedSource{}

	for _, sourceType := range ingest.Registered() {
		loadRules(store, ruleEngine, sourceType)

		cfgs, err := store.ListSources(sourceType)
		if err != nil {
			slog.Error("failed to load sources", "type", sourceType, "error", err)
			continue
		}
		var paths []string
		for _, c := range cfgs {
			paths = append(paths, c.Path)
		}

		src, ok := ingest.New(sourceType, paths, store, ruleEngine)
		if !ok {
			continue
		}
		scheduler.Register(src)
		managed[sourceType] = src
		slog.Info("source type initialized", "type", sourceType, "path_count", len(paths))
	}

	// Ingestion persistence — configurable buffer absorbs bursts (e.g. a
	// Minecraft server dumping thousands of boot-time log lines), configurable
	// batch size/interval controls write throughput to SQLite. Non-info events
	// are NOT individually logged here anymore — they're visible in the Watch
	// tab already; duplicating them into Activity was the exact mixing of
	// "monitored data" and "Heimdall's own operations" we're trying to avoid.
	persistCh := bus.Subscribe(cfg.EventBufferSize)
	go func() {
		batch := make([]core.Event, 0, cfg.BatchSize)
		ticker := time.NewTicker(cfg.BatchFlushInterval)
		defer ticker.Stop()

		flush := func() {
			if len(batch) == 0 {
				return
			}
			if err := store.SaveEvents(batch); err != nil {
				slog.Error("failed to save event batch", "count", len(batch), "error", err)
			} else {
				warnings, criticals := 0, 0
				for _, e := range batch {
					switch e.Severity {
					case "warning":
						warnings++
					case "critical":
						criticals++
					}
				}
				if warnings+criticals > 0 {
					slog.Info("event batch flushed", "total", len(batch), "warnings", warnings, "criticals", criticals)
				}
			}
			batch = batch[:0]
		}

		for {
			select {
			case e, ok := <-persistCh:
				if !ok {
					flush()
					return
				}
				batch = append(batch, e)
				if len(batch) >= cfg.BatchSize {
					flush()
				}
			case <-ticker.C:
				flush()
			}
		}
	}()

	// Activity log retention — prune anything older than the configured window.
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			activityLog.Prune(cfg.ActivityRetention)
		}
	}()

	reporter := reporting.New(store, bus, reporting.Config{
		OllamaURL: cfg.OllamaURL,
		Model:     cfg.LLMModel,
	})

	go func() {
		ticker := time.NewTicker(cfg.ReportInterval)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			if _, err := reporter.Generate(ctx, cfg.ReportInterval); err != nil {
				slog.Error("scheduled report generation failed", "error", err)
			}
			cancel()
		}
	}()
	slog.Info("report generation scheduled", "interval", cfg.ReportInterval)

	srv := api.New(bus, store, managed, ruleEngine, reporter, activityLog, authStore, sessions, ctl, status, cfg.SelfContainer)
	go func() {
		slog.Info("api server starting", "addr", cfg.APIAddr)
		if err := srv.Start(cfg.APIAddr); err != nil {
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

	status.Set("stopping")
	slog.Warn("shutdown signal received — stopping scheduler")
	close(stop)

	slog.Warn("allowing final event/activity batches to flush")
	time.Sleep(1500 * time.Millisecond)

	slog.Warn("shutdown complete")
}
