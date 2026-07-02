package storage

import (
	"database/sql"
	"time"
)

type RuleConfig struct {
	ID         int64
	SourceType string
	Pattern    string
	Severity   string
	EventType  string
	Priority   int
	Enabled    bool
	CreatedAt  time.Time
}

// ListRules returns rules for a source type, ordered so lower priority
// numbers are checked first. Empty sourceType returns rules for all types.
func (s *Store) ListRules(sourceType string) ([]RuleConfig, error) {
	var rows *sql.Rows
	var err error

	if sourceType == "" {
		rows, err = s.db.Query(`
			SELECT id, source_type, pattern, severity, event_type, priority, enabled, created_at
			FROM rules WHERE enabled = 1 ORDER BY source_type, priority ASC`)
	} else {
		rows, err = s.db.Query(`
			SELECT id, source_type, pattern, severity, event_type, priority, enabled, created_at
			FROM rules WHERE source_type = ? AND enabled = 1 ORDER BY priority ASC`, sourceType)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []RuleConfig{}
	for rows.Next() {
		var c RuleConfig
		var enabled int
		if err := rows.Scan(&c.ID, &c.SourceType, &c.Pattern, &c.Severity, &c.EventType, &c.Priority, &enabled, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.Enabled = enabled == 1
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) AddRule(sourceType, pattern, severity, eventType string, priority int) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO rules (source_type, pattern, severity, event_type, priority, enabled) VALUES (?, ?, ?, ?, ?, 1)`,
		sourceType, pattern, severity, eventType, priority,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) GetRule(id int64) (RuleConfig, error) {
	var c RuleConfig
	var enabled int
	err := s.db.QueryRow(
		`SELECT id, source_type, pattern, severity, event_type, priority, enabled, created_at FROM rules WHERE id = ?`, id,
	).Scan(&c.ID, &c.SourceType, &c.Pattern, &c.Severity, &c.EventType, &c.Priority, &enabled, &c.CreatedAt)
	c.Enabled = enabled == 1
	return c, err
}

func (s *Store) RemoveRule(id int64) error {
	_, err := s.db.Exec(`DELETE FROM rules WHERE id = ?`, id)
	return err
}
