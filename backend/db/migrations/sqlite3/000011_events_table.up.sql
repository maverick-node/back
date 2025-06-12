-- +migrate Up
CREATE TABLE
    IF NOT EXISTS events (
        id TEXT PRIMARY KEY,
        title TEXT NOT NULL,
        description TEXT NOT NULL,
        event_datetime DATETIME NOT NULL,
        location TEXT,
        creator_id INTEGER NOT NULL,
        group_id INTEGER NOT NULL,
        creation_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (creator_id) REFERENCES users (id),
        FOREIGN KEY (group_id) REFERENCES groups (id)
    );

-- +migrate Down
PRAGMA foreign_keys = OFF;

DROP TABLE IF EXISTS events;

PRAGMA foreign_keys = ON;