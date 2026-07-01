CREATE TABLE IF NOT EXISTS events (
                                      id        INTEGER PRIMARY KEY AUTOINCREMENT,
                                      timestamp DATETIME NOT NULL,
                                      source    TEXT NOT NULL,
                                      type      TEXT NOT NULL,
                                      severity  TEXT NOT NULL,
                                      message   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);

CREATE TABLE IF NOT EXISTS offsets (
                                       source TEXT NOT NULL,
                                       path   TEXT NOT NULL,
                                       offset INTEGER NOT NULL,
                                       PRIMARY KEY (source, path)
    );