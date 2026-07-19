package storage

import (
	"database/sql"
	"time"

	"heimdall/internal/core"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// --- events ---

func (s *Store) SaveEvent(e core.Event) error {
	_, err := s.db.Exec(
		`INSERT INTO events (timestamp, source, type, severity, message) VALUES (?, ?, ?, ?, ?)`,
		e.Timestamp, e.Source, e.Type, e.Severity, e.Message,
	)
	return err
}

func (s *Store) RecentEvents(limit int) ([]core.Event, error) {
	rows, err := s.db.Query(
		`SELECT timestamp, source, type, severity, message
		 FROM events ORDER BY id DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []core.Event{} // explicit empty slice, not nil — same fix as ListSources/ListRules
	for rows.Next() {
		var e core.Event
		var ts time.Time
		if err := rows.Scan(&ts, &e.Source, &e.Type, &e.Severity, &e.Message); err != nil {
			return nil, err
		}
		e.Timestamp = ts
		events = append(events, e)
	}
	return events, rows.Err()
}

// --- offsets ---

func (s *Store) GetOffset(source, path string) (int64, bool, error) {
	var offset int64
	err := s.db.QueryRow(
		`SELECT offset FROM offsets WHERE source = ? AND path = ?`, source, path,
	).Scan(&offset)

	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return offset, true, nil
}

func (s *Store) SetOffset(source, path string, offset int64) error {
	_, err := s.db.Exec(
		`INSERT INTO offsets (source, path, offset) VALUES (?, ?, ?)
		 ON CONFLICT(source, path) DO UPDATE SET offset = excluded.offset`,
		source, path, offset,
	)
	return err
}

// EventsSince returns events at or after the given time, oldest first.
func (s *Store) EventsSince(since time.Time) ([]core.Event, error) {
	rows, err := s.db.Query(
		`SELECT timestamp, source, type, severity, message FROM events WHERE timestamp >= ? ORDER BY id ASC`, since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []core.Event
	for rows.Next() {
		var e core.Event
		var ts time.Time
		if err := rows.Scan(&ts, &e.Source, &e.Type, &e.Severity, &e.Message); err != nil {
			return nil, err
		}
		e.Timestamp = ts
		events = append(events, e)
	}
	return events, rows.Err()
}
