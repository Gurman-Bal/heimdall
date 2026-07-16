package truenas

import (
	"strings"
	"time"

	"heimdall/internal/core"
	"heimdall/internal/ingest"
)

func init() {
	ingest.Register("truenas", ParseLine)
}

// ParseLine only extracts the raw event. Severity/Type are filled in by the
// rule engine after this returns - see main.go's classifier wiring.
func ParseLine(line string) core.Event {
	return core.Event{
		Timestamp: time.Now(),
		Source:    "truenas",
		Message:   strings.TrimSpace(line),
	}
}
