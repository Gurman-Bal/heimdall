package api

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"time"

	"heimdall/internal/core"
	"heimdall/internal/services/reporting"
	"heimdall/internal/storage"
)

//go:embed web/*
var webFS embed.FS

type ManagedSource interface {
	AddPath(path string)
	RemovePath(path string)
}

type Server struct {
	bus      *core.EventBus
	store    *storage.Store
	sources  map[string]ManagedSource
	rules    *core.RuleEngine
	reporter *reporting.Reporter
}

func New(bus *core.EventBus, store *storage.Store, sources map[string]ManagedSource, rules *core.RuleEngine, reporter *reporting.Reporter) *Server {
	return &Server{bus: bus, store: store, sources: sources, rules: rules, reporter: reporter}
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/events", s.handleEvents)
	mux.HandleFunc("/api/stream", s.handleStream)

	mux.HandleFunc("GET /api/sources", s.handleListSources)
	mux.HandleFunc("GET /api/source-types", s.handleSourceTypes)
	mux.HandleFunc("POST /api/sources", s.handleAddSource)
	mux.HandleFunc("DELETE /api/sources/{id}", s.handleDeleteSource)

	mux.HandleFunc("GET /api/rules", s.handleListRules)
	mux.HandleFunc("POST /api/rules", s.handleAddRule)
	mux.HandleFunc("DELETE /api/rules/{id}", s.handleDeleteRule)

	mux.HandleFunc("GET /api/reports", s.handleListReports)
	mux.HandleFunc("GET /api/reports/{id}", s.handleGetReport)
	mux.HandleFunc("POST /api/reports/generate", s.handleGenerateReport)

	static, err := fs.Sub(webFS, "web")
	if err != nil {
		return fmt.Errorf("failed to load embedded web assets: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(static)))

	return http.ListenAndServe(addr, mux)
}

// --- events ---

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.store.RecentEvents(200)
	if err != nil {
		http.Error(w, "failed to load events", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(events)
	if err != nil {
		return
	}
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := s.bus.Subscribe(20)
	for {
		select {
		case e := <-ch:
			data, _ := json.Marshal(e)
			_, err := fmt.Fprintf(w, "data: %s\n\n", data)
			if err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// --- sources ---

func (s *Server) handleListSources(w http.ResponseWriter, r *http.Request) {
	sourceType := r.URL.Query().Get("type") // empty = all types
	list, err := s.store.ListSources(sourceType)
	if err != nil {
		http.Error(w, "failed to load sources", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(list)
	if err != nil {
		return
	}
}

func (s *Server) handleSourceTypes(w http.ResponseWriter, r *http.Request) {
	types := make([]string, 0, len(s.sources))
	for t := range s.sources {
		types = append(types, t)
	}
	sort.Strings(types)
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(types)
	if err != nil {
		return
	}
}

type addSourceRequest struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

func (s *Server) handleAddSource(w http.ResponseWriter, r *http.Request) {
	var req addSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Type == "" || req.Path == "" {
		http.Error(w, "type and path are required", http.StatusBadRequest)
		return
	}

	managed, ok := s.sources[req.Type]
	if !ok {
		http.Error(w, fmt.Sprintf("unknown source type %q", req.Type), http.StatusBadRequest)
		return
	}

	id, err := s.store.AddSource(req.Type, req.Path)
	if err != nil {
		http.Error(w, "failed to save source", http.StatusInternalServerError)
		return
	}

	managed.AddPath(req.Path)
	slog.Info("source added via api", "type", req.Type, "path", req.Path, "id", id)

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]any{"id": id, "type": req.Type, "path": req.Path})
	if err != nil {
		return
	}
}

func (s *Server) handleDeleteSource(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	cfg, err := s.store.GetSource(id)
	if err != nil {
		http.Error(w, "source not found", http.StatusNotFound)
		return
	}

	if err := s.store.RemoveSource(id); err != nil {
		http.Error(w, "failed to remove source", http.StatusInternalServerError)
		return
	}

	if managed, ok := s.sources[cfg.Type]; ok {
		managed.RemovePath(cfg.Path)
	}
	slog.Info("source removed via api", "type", cfg.Type, "path", cfg.Path, "id", id)

	w.WriteHeader(http.StatusNoContent)
}

// --- rules ---

func (s *Server) handleListRules(w http.ResponseWriter, r *http.Request) {
	sourceType := r.URL.Query().Get("type")
	list, err := s.store.ListRules(sourceType)
	if err != nil {
		http.Error(w, "failed to load rules", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(list)
	if err != nil {
		return
	}
}

type addRuleRequest struct {
	Type      string `json:"type"`
	Pattern   string `json:"pattern"`
	Severity  string `json:"severity"`
	EventType string `json:"event_type"`
	Priority  int    `json:"priority"`
}

func (s *Server) handleAddRule(w http.ResponseWriter, r *http.Request) {
	var req addRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Type == "" || req.Pattern == "" || req.Severity == "" || req.EventType == "" {
		http.Error(w, "type, pattern, severity, and event_type are required", http.StatusBadRequest)
		return
	}
	if _, err := regexp.Compile(req.Pattern); err != nil {
		http.Error(w, fmt.Sprintf("invalid regex pattern: %v", err), http.StatusBadRequest)
		return
	}

	id, err := s.store.AddRule(req.Type, req.Pattern, req.Severity, req.EventType, req.Priority)
	if err != nil {
		http.Error(w, "failed to save rule", http.StatusInternalServerError)
		return
	}

	if errs := s.reloadRules(req.Type); len(errs) > 0 {
		slog.Warn("rule reload had errors", "type", req.Type, "errors", errs)
	}
	slog.Info("rule added via api", "type", req.Type, "pattern", req.Pattern, "id", id)

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]any{"id": id})
	if err != nil {
		return
	}
}

func (s *Server) handleDeleteRule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	cfg, err := s.store.GetRule(id)
	if err != nil {
		http.Error(w, "rule not found", http.StatusNotFound)
		return
	}
	if err := s.store.RemoveRule(id); err != nil {
		http.Error(w, "failed to remove rule", http.StatusInternalServerError)
		return
	}

	if errs := s.reloadRules(cfg.SourceType); len(errs) > 0 {
		slog.Warn("rule reload had errors", "type", cfg.SourceType, "errors", errs)
	}
	slog.Info("rule removed via api", "type", cfg.SourceType, "id", id)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) reloadRules(sourceType string) []error {
	cfgs, err := s.store.ListRules(sourceType)
	if err != nil {
		return []error{err}
	}
	defs := make([]core.RuleDef, len(cfgs))
	for i, c := range cfgs {
		defs[i] = core.RuleDef{ID: c.ID, Pattern: c.Pattern, Severity: c.Severity, EventType: c.EventType, Priority: c.Priority}
	}
	return s.rules.Load(sourceType, defs)
}

// --- reports ---

func (s *Server) handleListReports(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListReports(50)
	if err != nil {
		http.Error(w, "failed to load reports", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(list)
	if err != nil {
		return
	}
}

func (s *Server) handleGetReport(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	report, err := s.store.GetReport(id)
	if err != nil {
		http.Error(w, "report not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(report)
	if err != nil {
		return
	}
}

func (s *Server) handleGenerateReport(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	id, err := s.reporter.Generate(ctx, time.Hour)
	if err != nil {
		http.Error(w, fmt.Sprintf("report generation failed: %v", err), http.StatusInternalServerError)
		return
	}
	if id == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]any{"id": id})
	if err != nil {
		return
	}
}
