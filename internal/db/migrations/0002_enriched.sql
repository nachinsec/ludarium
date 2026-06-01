-- +goose Up
ALTER TABLE games ADD COLUMN enriched_at TEXT;

-- +goose Down
ALTER TABLE games DROP COLUMN enriched_at;
