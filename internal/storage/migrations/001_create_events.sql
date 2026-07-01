-- +goose Up

CREATE TABLE events (
                        id INTEGER PRIMARY KEY,
                        timestamp TEXT,
                        source TEXT,
                        type TEXT,
                        severity TEXT,
                        message TEXT
);

-- +goose Down

DROP TABLE events;