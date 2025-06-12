-- +migrate Up
CREATE TABLE
    IF NOT EXISTS messages (
        id TEXT PRIMARY KEY,
        sender_id INTEGER NOT NULL,
        receiver_id INTEGER NOT NULL,
        content TEXT NOT NULL,
        creation_date DATETIME NOT NULL,
        FOREIGN KEY (sender_id) REFERENCES users (id),
        FOREIGN KEY (receiver_id) REFERENCES users (id)
    );

-- +migrate Down
PRAGMA foreign_keys = OFF;

DROP TABLE IF EXISTS messages;

PRAGMA foreign_keys = ON;