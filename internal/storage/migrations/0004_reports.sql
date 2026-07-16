CREATE TABLE IF NOT EXISTS reports (
                                       id           INTEGER PRIMARY KEY AUTOINCREMENT,
                                       generated_at DATETIME NOT NULL,
                                       period_start DATETIME NOT NULL,
                                       period_end   DATETIME NOT NULL,
                                       event_count  INTEGER NOT NULL,
                                       summary      TEXT NOT NULL,
                                       issues_json  TEXT NOT NULL,
                                       model        TEXT NOT NULL
);