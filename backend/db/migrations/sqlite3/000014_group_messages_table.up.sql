-- +migrate Up
CREATE TABLE
    IF NOT EXISTS group_messages (
        id TEXT PRIMARY KEY,
        group_id TEXT NOT NULL,
        sender_id TEXT NOT NULL,
        content TEXT NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (group_id) REFERENCES groups (id) ON DELETE CASCADE,
        FOREIGN KEY (sender_id) REFERENCES users (id) ON DELETE CASCADE
    );

CREATE INDEX IF NOT EXISTS idx_group_messages_group_id ON group_messages (group_id);

CREATE INDEX IF NOT EXISTS idx_group_messages_created_at ON group_messages (created_at);

-- +migrate Down
PRAGMA foreign_keys = OFF;

DROP TABLE IF EXISTS group_messages;

PRAGMA foreign_keys = ON;