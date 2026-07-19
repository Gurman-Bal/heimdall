package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"heimdall/internal/api"
	"heimdall/internal/config"
	"heimdall/internal/core"
	"heimdall/internal/ingest"
	_ "heimdall/internal/plugins/minecraft"
	_ "heimdall/internal/plugins/truenas"
	"heimdall/internal/services/reporting"
	"heimdall/internal/storage"
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
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.Load()
	slog.Info("config loaded",
		"db_path", cfg.DBPath, "log_dir", cfg.DefaultLogDir, "api_addr", cfg.APIAddr,
		"ollama_url", cfg.OllamaURL, "llm_model", cfg.LLMModel, "report_interval", cfg.ReportInterval)

	store, err := storage.New(cfg.DBPath)
	if err != nil {
		slog.Error("failed to open storage", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	slog.Info("storage opened", "path", cfg.DBPath)

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

	srv := api.New(bus, store, managed, ruleEngine, reporter)
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
	slog.Info("shutting down")
	close(stop)
}
