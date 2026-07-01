package api

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"

	"heimdall/internal/core"
	"heimdall/internal/storage"
)

//go:embed web/*
var webFS embed.FS

type ManagedSource interface {
	AddPath(path string)
	RemovePath(path string)
}

type Server struct {
	bus     *core.EventBus
	store   *storage.Store
	sources map[string]ManagedSource
}

func New(bus *core.EventBus, store *storage.Store, sources map[string]ManagedSource) *Server {
	return &Server{bus: bus, store: store, sources: sources}
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/events", s.handleEvents)
	mux.HandleFunc("/api/stream", s.handleStream)
	mux.HandleFunc("GET /api/sources", s.handleListSources)
	mux.HandleFunc("POST /api/sources", s.handleAddSource)
	mux.HandleFunc("DELETE /api/sources/{id}", s.handleDeleteSource)

	static, err := fs.Sub(webFS, "web")
	if err != nil {
		return fmt.Errorf("failed to load embedded web assets: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(static)))

	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.store.RecentEvents(200)
	if err != nil {
		http.Error(w, "failed to load events", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
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
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) handleListSources(w http.ResponseWriter, r *http.Request) {
	sourceType := r.URL.Query().Get("type")
	if sourceType == "" {
		sourceType = "truenas"
	}

	list, err := s.store.ListSources(sourceType)
	if err != nil {
		http.Error(w, "failed to load sources", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"id": id, "type": req.Type, "path": req.Path})
}

func (s *Server) handleDeleteSource(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
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

	w.WriteHeader(http.StatusNoContent)
}
