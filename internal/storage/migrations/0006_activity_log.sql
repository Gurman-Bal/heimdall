-- +goose Up
CREATE TABLE IF NOT EXISTS activity_log (
                                            id      INTEGER PRIMARY KEY AUTOINCREMENT,
                                            time    DATETIME NOT NULL,
                                            level   TEXT NOT NULL,
                                            message TEXT NOT NULL,
                                            attrs   TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_activity_log_time ON activity_log(time);

-- +goose Down
DROP TABLE IF EXISTS activity_log;