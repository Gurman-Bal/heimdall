package core

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type ActivityEntry struct {
	Time    time.Time         `json:"time"`
	Level   string            `json:"level"`
	Message string            `json:"message"`
	Attrs   map[string]string `json:"attrs"`
}

// ActivityLog is a fixed-size ring buffer of Heimdall's own log output,
// viewable from the dashboard without needing terminal/docker access.
type ActivityLog struct {
	mu      sync.Mutex
	entries []ActivityEntry
	max     int
}

func NewActivityLog(max int) *ActivityLog {
	return &ActivityLog{max: max}
}

func (a *ActivityLog) add(e ActivityEntry) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, e)
	if len(a.entries) > a.max {
		a.entries = a.entries[len(a.entries)-a.max:]
	}
}

// Recent returns entries oldest-first, newest last, never nil.
func (a *ActivityLog) Recent() []ActivityEntry {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]ActivityEntry, len(a.entries))
	copy(out, a.entries)
	return out
}

// Handler wraps an existing slog.Handler, mirroring every record into this
// ActivityLog in addition to whatever the wrapped handler does (stdout, etc).
func (a *ActivityLog) Handler(wrapped slog.Handler) slog.Handler {
	return &activityHandler{wrapped: wrapped, log: a}
}

type activityHandler struct {
	wrapped slog.Handler
	log     *ActivityLog
}

func (h *activityHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.wrapped.Enabled(ctx, level)
}

func (h *activityHandler) Handle(ctx context.Context, r slog.Record) error {
	attrs := map[string]string{}
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.String()
		return true
	})
	h.log.add(ActivityEntry{Time: r.Time, Level: r.Level.String(), Message: r.Message, Attrs: attrs})
	return h.wrapped.Handle(ctx, r)
}

func (h *activityHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &activityHandler{wrapped: h.wrapped.WithAttrs(attrs), log: h.log}
}

func (h *activityHandler) WithGroup(name string) slog.Handler {
	return &activityHandler{wrapped: h.wrapped.WithGroup(name), log: h.log}
}
