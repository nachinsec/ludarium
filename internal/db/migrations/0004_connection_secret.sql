-- +goose Up
ALTER TABLE connections ADD COLUMN secret TEXT NOT NULL DEFAULT ''; -- AES-GCM encrypted (e.g. PSN refresh token)

-- +goose Down
ALTER TABLE connections DROP COLUMN secret;
