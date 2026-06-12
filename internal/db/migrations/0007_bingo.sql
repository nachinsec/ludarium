-- +goose Up
-- Conference bingo boards. The 5x5 grid + marks live as JSON in `data`,
-- so the server just stores the document and the frontend owns its shape.
CREATE TABLE bingo_boards (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title      TEXT NOT NULL DEFAULT 'Bingo',
    data       TEXT NOT NULL DEFAULT '{}',
    visibility TEXT NOT NULL DEFAULT 'private', -- 'private' | 'public'
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_bingo_user ON bingo_boards(user_id);

-- +goose Down
DROP TABLE bingo_boards;
