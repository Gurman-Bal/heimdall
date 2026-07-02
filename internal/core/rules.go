package core

import (
	"fmt"
	"regexp"
	"sort"
	"sync"
)

// RuleDef is the plain-data form the engine compiles from. Callers (main.go,
// the api package) build these from whatever storage returns.
type RuleDef struct {
	ID        int64
	Pattern   string
	Severity  string
	EventType string
	Priority  int
}

type compiledRule struct {
	id        int64
	re        *regexp.Regexp
	severity  string
	eventType string
	priority  int
}

// RuleEngine classifies raw log lines into severity/type based on rules
// loaded per source type. Safe for concurrent use: Load can be called from
// the API while Classify is being called from the polling loop.
type RuleEngine struct {
	mu    sync.RWMutex
	rules map[string][]compiledRule
}

func NewRuleEngine() *RuleEngine {
	return &RuleEngine{rules: map[string][]compiledRule{}}
}

// Load compiles and installs the rule set for a source type, replacing
// whatever was loaded before for that type. Rules with invalid regex are
// skipped; their errors are returned so the caller can log/report them
// instead of silently losing a bad rule.
func (re *RuleEngine) Load(sourceType string, defs []RuleDef) []error {
	compiled := make([]compiledRule, 0, len(defs))
	var errs []error

	for _, d := range defs {
		pattern, err := regexp.Compile(d.Pattern)
		if err != nil {
			errs = append(errs, fmt.Errorf("rule %d (%q): %w", d.ID, d.Pattern, err))
			continue
		}
		compiled = append(compiled, compiledRule{
			id: d.ID, re: pattern, severity: d.Severity, eventType: d.EventType, priority: d.Priority,
		})
	}

	sort.Slice(compiled, func(i, j int) bool { return compiled[i].priority < compiled[j].priority })

	re.mu.Lock()
	re.rules[sourceType] = compiled
	re.mu.Unlock()

	return errs
}

// Classify returns severity/type for a raw line: first matching rule wins,
// lowest priority number checked first. Falls back to info/log.
func (re *RuleEngine) Classify(sourceType, message string) (severity, eventType string) {
	re.mu.RLock()
	rules := re.rules[sourceType]
	re.mu.RUnlock()

	for _, r := range rules {
		if r.re.MatchString(message) {
			return r.severity, r.eventType
		}
	}
	return "info", "log"
}
