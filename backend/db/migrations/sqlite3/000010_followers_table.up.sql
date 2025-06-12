-- +migrate Up
CREATE TABLE
    IF NOT EXISTS Followers (
        id TEXT PRIMARY KEY,
        follower_id INTEGER NOT NULL,
        followed_id INTEGER NOT NULL,
        status TEXT NOT NULL,
        FOREIGN KEY (follower_id) REFERENCES users (id),
        FOREIGN KEY (followed_id) REFERENCES users (id)
    );

-- +migrate Down
PRAGMA foreign_keys = OFF;

DROP TABLE IF EXISTS Followers;

PRAGMA foreign_keys = ON;