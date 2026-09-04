package truenas

import (
	"strings"
	"time"

	"heimdall/internal/core"
	"heimdall/internal/ingest"
)

func init() {
	ingest.Register("truenas", ParseLine)
	ingest.RegisterDefaultRules("truenas", []ingest.DefaultRule{
		{Pattern: `(?i)\b(reallocated sector|pending sector|smart.*fail)\b`, Severity: "critical", EventType: "smart_warning"},
		{Pattern: `(?i)\b(panic|critical|failed|failure)\b`, Severity: "critical", EventType: "error"},
		{Pattern: `(?i)\b(degraded|warn|warning)\b`, Severity: "warning", EventType: "warning"},
		{Pattern: `(?i)\b(denied|refused|error)\b`, Severity: "warning", EventType: "error"},
	})
}

func ParseLine(line string) core.Event {
	return core.Event{
		Timestamp: time.Now(),
		Source:    "truenas",
		Message:   strings.TrimSpace(line),
	}
}
