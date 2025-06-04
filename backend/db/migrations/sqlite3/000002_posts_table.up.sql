-- +migrate Up
CREATE TABLE
    IF NOT EXISTS posts (
        id TEXT PRIMARY KEY,
        user_id TEXT NOT NULL,
        author TEXT,
        title TEXT NOT NULL,
        content TEXT NOT NULL,
        image TEXT DEFAULT '',
        creation_date DATETIME NOT NULL,
        status TEXT NOT NULL,
        FOREIGN KEY (user_id) REFERENCES users (id)
    );