-- +migrate Up
CREATE TABLE
    IF NOT EXISTS postsPrivacy (
        id TEXT PRIMARY KEY,
        post_id INTEGER NOT NULL,
        user_id INTEGER NOT NULL,
        FOREIGN KEY (post_id) REFERENCES posts (id),
        FOREIGN KEY (user_id) REFERENCES users (id),
        UNIQUE (post_id, user_id)
    );

