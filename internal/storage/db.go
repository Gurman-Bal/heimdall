package storage

import (
	"database/sql"
	"fmt"
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

	// SQLite is shared by the controller and worker processes.
	//
	// WAL allows readers to continue while another process is writing.
	// busy_timeout gives SQLite time to wait for a competing writer rather
	// than immediately returning "database is locked".
	//
	// Keep one connection per process. This makes connection-specific SQLite
	// PRAGMAs such as busy_timeout predictable and avoids unnecessary
	// concurrent connections against the same SQLite database.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	if _, err := db.Exec(`PRAGMA busy_timeout=5000;`); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
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

// -----------------------------------------------------------------------------
// Events
// -----------------------------------------------------------------------------

func (s *Store) SaveEvent(e core.Event) error {
	_, err := s.db.Exec(
		`INSERT INTO events
(timestamp, source, type, severity, message)
VALUES (?, ?, ?, ?, ?)`,
		e.Timestamp,
		e.Source,
		e.Type,
		e.Severity,
		e.Message,
	)

	return err
}

func (s *Store) RecentEvents(limit int) ([]core.Event, error) {
	rows, err := s.db.Query(
		`SELECT timestamp, source, type, severity, message
FROM events
ORDER BY id DESC
LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Return [] rather than null when there are no events.
	events := []core.Event{}

	for rows.Next() {
		var e core.Event
		var ts time.Time

		if err := rows.Scan(
			&ts,
			&e.Source,
			&e.Type,
			&e.Severity,
			&e.Message,
		); err != nil {
			return nil, err
		}

		e.Timestamp = ts
		events = append(events, e)
	}

	return events, rows.Err()
}

// -----------------------------------------------------------------------------
// Offsets
// -----------------------------------------------------------------------------

func (s *Store) GetOffset(source, path string) (int64, bool, error) {
	var offset int64

	err := s.db.QueryRow(
		`SELECT offset
FROM offsets
WHERE source = ? AND path = ?`,
		source,
		path,
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
		`INSERT INTO offsets (source, path, offset)
VALUES (?, ?, ?)
ON CONFLICT(source, path)
DO UPDATE SET offset = excluded.offset`,
		source,
		path,
		offset,
	)

	return err
}

// EventsSince returns events at or after the given time, oldest first.
func (s *Store) EventsSince(since time.Time) ([]core.Event, error) {
	rows, err := s.db.Query(
		`SELECT timestamp, source, type, severity, message
FROM events
WHERE timestamp >= ?
ORDER BY id ASC`,
		since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []core.Event{}

	for rows.Next() {
		var e core.Event
		var ts time.Time

		if err := rows.Scan(
			&ts,
			&e.Source,
			&e.Type,
			&e.Severity,
			&e.Message,
		); err != nil {
			return nil, err
		}

		e.Timestamp = ts
		events = append(events, e)
	}

	return events, rows.Err()
}

func (s *Store) SaveEvents(events []core.Event) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(
		`INSERT INTO events
(timestamp, source, type, severity, message)
VALUES (?, ?, ?, ?, ?)`,
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, e := range events {
		if _, err := stmt.Exec(
			e.Timestamp,
			e.Source,
			e.Type,
			e.Severity,
			e.Message,
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
