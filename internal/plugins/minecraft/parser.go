package minecraft

import (
	"strings"
	"time"

	"heimdall/internal/core"
	"heimdall/internal/ingest"
)

func init() {
	ingest.Register("minecraft", ParseLine)
	ingest.RegisterDefaultRules("minecraft", []ingest.DefaultRule{
		{Pattern: `(?i)\bOutOfMemoryError\b`, Severity: "critical", EventType: "crash"},
		{Pattern: `(?i)(Exception in server tick loop|server thread/FATAL)`, Severity: "critical", EventType: "crash"},
		{Pattern: `(?i)Can't keep up! Is the server overloaded`, Severity: "warning", EventType: "tps_warning"},
		{Pattern: `(?i)Lithium Class Analysis Error`, Severity: "info", EventType: "lithium_noise"},
		{Pattern: `(?i)\b(ERROR|Exception)\b`, Severity: "warning", EventType: "error"},
		{Pattern: `(?i)joined the game`, Severity: "info", EventType: "player_join"},
		{Pattern: `(?i)left the game`, Severity: "info", EventType: "player_leave"},
	})
}

func ParseLine(line string) core.Event {
	return core.Event{
		Timestamp: time.Now(),
		Source:    "minecraft",
		Message:   strings.TrimSpace(line),
	}
}
