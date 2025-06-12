-- +migrate Up
CREATE TABLE
    IF NOT EXISTS comments (
        id TEXT PRIMARY KEY NOT NULL,
        post_id TEXT NOT NULL,
        author TEXT NOT NULL,
        content TEXT NOT NULL,
        image TEXT DEFAULT '',
        creation_date DATETIME NOT NULL,
        FOREIGN KEY (post_id) REFERENCES posts (id)
    );

-- +migrate Down
PRAGMA foreign_keys = OFF;

DROP TABLE IF EXISTS comments;

PRAGMA foreign_keys = ON;