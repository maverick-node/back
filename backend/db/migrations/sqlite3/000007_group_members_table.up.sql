-- +migrate Up
CREATE TABLE
    IF NOT EXISTS group_members (
        group_id TEXT NOT NULL,
        user_id INTEGER NOT NULL,
        status TEXT NOT NULL,
        is_admin INTEGER NOT NULL,
        FOREIGN KEY (group_id) REFERENCES groups (id),
        FOREIGN KEY (user_id) REFERENCES users (id)
    );
