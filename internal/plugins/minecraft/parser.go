package minecraft

import (
	"strings"
	"time"

	"heimdall/internal/core"
	"heimdall/internal/ingest"
)

func init() {
	ingest.Register("minecraft", ParseLine)
}

func ParseLine(line string) core.Event {
	return core.Event{
		Timestamp: time.Now(),
		Source:    "minecraft",
		Message:   strings.TrimSpace(line),
	}
}
