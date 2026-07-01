package minecraft

import (
	"regexp"
	"strings"
	"time"

	"heimdall/internal/core"
	"heimdall/internal/ingest"
)

func init() {
	ingest.Register("minecraft", ParseLine)
}

var rules = []struct {
	pattern  *regexp.Regexp
	severity string
	typ      string
}{
	{regexp.MustCompile(`(?i)\bOutOfMemoryError\b`), "critical", "crash"},
	{regexp.MustCompile(`(?i)(Exception in server tick loop|server thread/FATAL)`), "critical", "crash"},
	{regexp.MustCompile(`(?i)Can't keep up! Is the server overloaded`), "warning", "tps_warning"},
	{regexp.MustCompile(`(?i)\b(ERROR|Exception)\b`), "warning", "error"},
	{regexp.MustCompile(`(?i)joined the game`), "info", "player_join"},
	{regexp.MustCompile(`(?i)left the game`), "info", "player_leave"},
}

func ParseLine(line string) core.Event {
	severity, typ := "info", "log"
	for _, r := range rules {
		if r.pattern.MatchString(line) {
			severity, typ = r.severity, r.typ
			break
		}
	}
	return core.Event{
		Timestamp: time.Now(),
		Source:    "minecraft",
		Type:      typ,
		Severity:  severity,
		Message:   strings.TrimSpace(line),
	}
}
