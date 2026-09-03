package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"
)

var processStartedAt = time.Now()

func (s *Server) runCommand(ctx context.Context, raw string) error {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(raw)))
	if len(fields) == 0 {
		return fmt.Errorf("empty command")
	}

	verb := fields[0]
	target := s.selfContainer
	if len(fields) > 1 {
		target = fields[1]
	}
	if len(fields) > 2 {
		return fmt.Errorf("too many arguments")
	}

	switch verb {
	case "restart":
		s.status.Set("restarting")
		slog.Warn("restart requested", "target", target)
		err := s.dockerctl.Restart(ctx, target)
		if err != nil {
			s.status.Set("running")
		}
		return err
	case "stop", "shutdown":
		s.status.Set("stopping")
		slog.Warn("stop requested", "target", target)
		return s.dockerctl.Stop(ctx, target)
	case "start":
		slog.Info("start requested", "target", target)
		err := s.dockerctl.Start(ctx, target)
		if err == nil {
			s.status.Set("running")
		}
		return err
	default:
		return fmt.Errorf("unknown command %q — allowed: restart, stop, shutdown, start [target]", verb)
	}
}

type commandRequest struct {
	Command string `json:"command"`
}

func (s *Server) handleSystemCommand(w http.ResponseWriter, r *http.Request) {
	var req commandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := s.runCommand(ctx, req.Command); err != nil {
		slog.Warn("system command rejected or failed", "command", req.Command, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	slog.Info("system command executed", "command", req.Command)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleListContainers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.dockerctl.Allowed())
}

func (s *Server) handleContainerAction(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	action := r.PathValue("action")

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := s.runCommand(ctx, action+" "+name); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.auth.ChangePassword(req.CurrentPassword, req.NewPassword); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	types := make([]string, 0, len(s.sources))
	for t := range s.sources {
		types = append(types, t)
	}
	sort.Strings(types)

	status := map[string]any{
		"state":            s.status.Get(),
		"uptime_seconds":   int(time.Since(processStartedAt).Seconds()),
		"registered_types": types,
		"events_dropped":   s.bus.DroppedCount(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleActivity(w http.ResponseWriter, r *http.Request) {
	since := time.Now().Add(-24 * time.Hour)
	if v := r.URL.Query().Get("since"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			since = time.Now().Add(-d)
		}
	}

	entries, err := s.activityLog.Recent(since, 1000)
	if err != nil {
		http.Error(w, "failed to load activity", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}
