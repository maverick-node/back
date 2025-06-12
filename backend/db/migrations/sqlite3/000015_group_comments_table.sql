-- +migrate Up
CREATE TABLE
    IF NOT EXISTS group_comments (
        id TEXT PRIMARY KEY NOT NULL,
        group_post_id TEXT NOT NULL,
        author TEXT NOT NULL,
        content TEXT NOT NULL,
        creation_date DATETIME NOT NULL,
        image TEXT DEFAULT '',
        FOREIGN KEY (group_post_id) REFERENCES group_posts (id)
    );

-- +migrate Down
PRAGMA foreign_keys = OFF;

DROP TABLE IF EXISTS group_comments;

PRAGMA foreign_keys = ON;