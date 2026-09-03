package storage

import (
	"encoding/json"
	"time"

	"heimdall/internal/core"
)

func (s *Store) SaveActivity(e core.ActivityEntry) error {
	attrsJSON, err := json.Marshal(e.Attrs)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO activity_log (time, level, message, attrs) VALUES (?, ?, ?, ?)`,
		e.Time, e.Level, e.Message, string(attrsJSON),
	)
	return err
}

func (s *Store) RecentActivity(since time.Time, limit int) ([]core.ActivityEntry, error) {
	rows, err := s.db.Query(
		`SELECT time, level, message, attrs FROM activity_log WHERE time >= ? ORDER BY id DESC LIMIT ?`,
		since, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []core.ActivityEntry{}
	for rows.Next() {
		var e core.ActivityEntry
		var attrsJSON string
		if err := rows.Scan(&e.Time, &e.Level, &e.Message, &attrsJSON); err != nil {
			return nil, err
		}
		e.Attrs = map[string]string{}
		json.Unmarshal([]byte(attrsJSON), &e.Attrs)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) PruneActivityOlderThan(cutoff time.Time) (int64, error) {
	res, err := s.db.Exec(`DELETE FROM activity_log WHERE time < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
