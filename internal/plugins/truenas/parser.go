package truenas

import (
	"regexp"
	"strings"
	"time"

	"heimdall/internal/core"
)

var rules = []struct {
	pattern  *regexp.Regexp
	severity string
	typ      string
}{
	{regexp.MustCompile(`(?i)\b(reallocated sector|pending sector|smart.*fail)\b`), "critical", "smart_warning"},
	{regexp.MustCompile(`(?i)\b(panic|critical|failed|failure)\b`), "critical", "error"},
	{regexp.MustCompile(`(?i)\b(degraded|warn|warning)\b`), "warning", "warning"},
	{regexp.MustCompile(`(?i)\b(denied|refused|error)\b`), "warning", "error"},
}

func parseLine(line string) core.Event {
	severity, typ := "info", "log"

	for _, r := range rules {
		if r.pattern.MatchString(line) {
			severity, typ = r.severity, r.typ
			break
		}
	}

	return core.Event{
		Timestamp: time.Now(),
		Source:    "truenas",
		Type:      typ,
		Severity:  severity,
		Message:   strings.TrimSpace(line),
	}
}
