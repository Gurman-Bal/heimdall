package core

import "time"

type Event struct {
	Timestamp time.Time
	Source    string
	Type      string
	Severity  string
	Message   string
}
