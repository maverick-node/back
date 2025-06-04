-- +migrate Up
CREATE TABLE
    IF NOT EXISTS event_responses (
        id TEXT PRIMARY KEY,
        user_id INTEGER NOT NULL,
        event_id INTEGER NOT NULL,
        option INTEGER NOT NULL, -- 1 for "Yes", -1 for "No" 
        response_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES users (id),
        FOREIGN KEY (event_id) REFERENCES events (id),
        UNIQUE (user_id, event_id)
    );
