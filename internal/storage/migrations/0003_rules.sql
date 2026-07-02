CREATE TABLE IF NOT EXISTS rules (
                                     id          INTEGER PRIMARY KEY AUTOINCREMENT,
                                     source_type TEXT NOT NULL,
                                     pattern     TEXT NOT NULL,
                                     severity    TEXT NOT NULL,
                                     event_type  TEXT NOT NULL,
                                     priority    INTEGER NOT NULL DEFAULT 100,
                                     enabled     INTEGER NOT NULL DEFAULT 1,
                                     created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_rules_source_type ON rules(source_type);