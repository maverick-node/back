-- +migrate Up
PRAGMA foreign_keys = ON;
CREATE TABLE
    IF NOT EXISTS users (
        id TEXT PRIMARY KEY NOT NULL,
        username TEXT NOT NULL UNIQUE,
        email TEXT NOT NULL UNIQUE,
        password TEXT NOT NULL,
        first_name TEXT NOT NULL,
        last_name TEXT NOT NULL,
        nickname TEXT DEFAULT '',
        bio TEXT DEFAULT '',
        date_of_birth TEXT NOT NULL,
        privacy TEXT NOT NULL DEFAULT 'public',
        avatar TEXT DEFAULT ''
    );