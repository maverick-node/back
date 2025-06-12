-- +migrate Up
CREATE TABLE
    IF NOT EXISTS group_posts (
        id TEXT PRIMARY KEY,
        group_id TEXT NOT NULL,
        user_id TEXT NOT NULL,
        title TEXT NOT NULL,
        content TEXT NOT NULL,
        image TEXT DEFAULT "",
        creation_date DATETIME NOT NULL,
        FOREIGN KEY (group_id) REFERENCES groups (id),
        FOREIGN KEY (user_id) REFERENCES users (id)
    );

-- +migrate Down
PRAGMA foreign_keys = OFF;

DROP TABLE IF EXISTS group_posts;

PRAGMA foreign_keys = ON;