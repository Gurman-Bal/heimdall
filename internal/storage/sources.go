package storage

import "time"

type SourceConfig struct {
	ID        int64
	Type      string
	Path      string
	Enabled   bool
	CreatedAt time.Time
}

func (s *Store) ListSources(sourceType string) ([]SourceConfig, error) {
	rows, err := s.db.Query(
		`SELECT id, type, path, enabled, created_at FROM sources WHERE type = ? AND enabled = 1`,
		sourceType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SourceConfig
	for rows.Next() {
		var c SourceConfig
		var enabled int
		if err := rows.Scan(&c.ID, &c.Type, &c.Path, &enabled, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.Enabled = enabled == 1
		out = append(out, c)
	}
	return out, rows.Err()
}

// AddSource inserts a new source, or re-enables it if the (type, path) pair
// already exists but was previously removed.
func (s *Store) AddSource(sourceType, path string) (int64, error) {
	_, err := s.db.Exec(
		`INSERT INTO sources (type, path, enabled) VALUES (?, ?, 1)
		 ON CONFLICT(type, path) DO UPDATE SET enabled = 1`,
		sourceType, path,
	)
	if err != nil {
		return 0, err
	}

	var id int64
	err = s.db.QueryRow(`SELECT id FROM sources WHERE type = ? AND path = ?`, sourceType, path).Scan(&id)
	return id, err
}

func (s *Store) GetSource(id int64) (SourceConfig, error) {
	var c SourceConfig
	var enabled int
	err := s.db.QueryRow(
		`SELECT id, type, path, enabled, created_at FROM sources WHERE id = ?`, id,
	).Scan(&c.ID, &c.Type, &c.Path, &enabled, &c.CreatedAt)
	c.Enabled = enabled == 1
	return c, err
}

func (s *Store) RemoveSource(id int64) error {
	_, err := s.db.Exec(`DELETE FROM sources WHERE id = ?`, id)
	return err
}
