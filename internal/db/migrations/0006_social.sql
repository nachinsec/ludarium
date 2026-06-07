-- +goose Up
-- Per-profile visibility: private by default (privacy-first; users opt in to sharing).
ALTER TABLE users ADD COLUMN visibility TEXT NOT NULL DEFAULT 'private'; -- 'private' | 'public'

-- Asymmetric follows: follower_id follows following_id (no approval needed).
CREATE TABLE follows (
    follower_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    following_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (follower_id, following_id)
);
CREATE INDEX idx_follows_following ON follows(following_id);

-- +goose Down
DROP TABLE follows;
ALTER TABLE users DROP COLUMN visibility;
