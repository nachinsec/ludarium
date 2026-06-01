-- +goose Up

-- A first-class account. Identity lives here; Steam is just one way to log in.
CREATE TABLE users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT    NOT NULL UNIQUE,            -- handle, used to log in
    display_name  TEXT    NOT NULL DEFAULT '',        -- shown name (defaults to username)
    email         TEXT    UNIQUE,                     -- nullable: steam-only accounts may lack one
    password_hash TEXT    NOT NULL DEFAULT '',        -- empty = no password set (steam-only)
    avatar_url    TEXT    NOT NULL DEFAULT '',
    role          TEXT    NOT NULL DEFAULT 'user',    -- 'user' | 'instance_admin'
    created_at    TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at    TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- External accounts linked to a user (Steam today, PSN/Xbox/etc. tomorrow).
CREATE TABLE connections (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider    TEXT    NOT NULL,                     -- 'steam'
    external_id TEXT    NOT NULL,                     -- e.g. steamid64
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    UNIQUE(provider, external_id)
);
CREATE INDEX idx_connections_user ON connections(user_id);

CREATE TABLE sessions (
    id         TEXT    PRIMARY KEY,        -- opaque random token
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT    NOT NULL DEFAULT (datetime('now')),
    expires_at TEXT    NOT NULL
);
CREATE INDEX idx_sessions_user ON sessions(user_id);

-- Global, shared metadata cache. One row per game regardless of how many
-- users have it in their library.
CREATE TABLE games (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    igdb_id      INTEGER UNIQUE,
    steam_appid  INTEGER UNIQUE,
    title        TEXT    NOT NULL,
    cover_url    TEXT    NOT NULL DEFAULT '',
    release_year INTEGER,
    developer    TEXT    NOT NULL DEFAULT '',
    genres       TEXT    NOT NULL DEFAULT '[]',  -- JSON array
    platforms    TEXT    NOT NULL DEFAULT '[]',  -- JSON array
    created_at   TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at   TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- A user's personal relationship to a game.
CREATE TABLE library_entries (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    game_id     INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    status      TEXT    NOT NULL DEFAULT 'backlog', -- backlog|playing|completed|dropped|wishlist
    rating      INTEGER,                            -- 1..5, nullable
    hours       REAL    NOT NULL DEFAULT 0,
    platform    TEXT    NOT NULL DEFAULT '',        -- where the user owns/plays it
    notes       TEXT    NOT NULL DEFAULT '',
    started_at  TEXT,
    finished_at TEXT,
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, game_id)
);
CREATE INDEX idx_library_user ON library_entries(user_id);
CREATE INDEX idx_library_status ON library_entries(user_id, status);

CREATE TABLE sync_jobs (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source     TEXT    NOT NULL,            -- 'steam'
    status     TEXT    NOT NULL DEFAULT 'pending', -- pending|running|done|error
    error      TEXT    NOT NULL DEFAULT '',
    last_run   TEXT,
    created_at TEXT    NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_sync_user ON sync_jobs(user_id);

-- +goose Down
DROP TABLE sync_jobs;
DROP TABLE library_entries;
DROP TABLE games;
DROP TABLE sessions;
DROP TABLE connections;
DROP TABLE users;
