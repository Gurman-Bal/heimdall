CREATE TABLE IF NOT EXISTS sources (
                                       id         INTEGER PRIMARY KEY AUTOINCREMENT,
                                       type       TEXT NOT NULL,
                                       path       TEXT NOT NULL,
                                       enabled    INTEGER NOT NULL DEFAULT 1,
                                       created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                                       UNIQUE(type, path)
    );