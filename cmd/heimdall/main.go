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
	_ "heimdall/internal/plugins/minecraft"
	_ "heimdall/internal/plugins/truenas"
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
		return // already seeded or user-customized, leave alone
	}

	for i, r := range defaultRules[sourceType] {
		priority := (i + 1) * 10
		if _, err := store.AddRule(sourceType, r.pattern, r.severity, r.eventType, priority); err != nil {
			slog.Error("failed to seed rule", "type", sourceType, "pattern", r.pattern, "error", err)
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

	srv := api.New(bus, store, managed, ruleEngine)
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
