-- +migrate Up
CREATE TABLE
    IF NOT EXISTS groups (
        id TEXT PRIMARY KEY,
        creator_id INTEGER NOT NULL,
        title TEXT NOT NULL,
        description TEXT,
        FOREIGN KEY (creator_id) REFERENCES users (id)
    );

-- +migrate Down
PRAGMA foreign_keys = OFF;

DROP TABLE IF EXISTS groups;

PRAGMA foreign_keys = ON;