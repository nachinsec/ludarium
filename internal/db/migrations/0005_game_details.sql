-- +goose Up
ALTER TABLE games ADD COLUMN details TEXT NOT NULL DEFAULT '{}'; -- JSON: summary, screenshots, score
-- Clear the enrich marker so existing Steam games re-fetch the new fields.
UPDATE games SET enriched_at = NULL WHERE steam_appid IS NOT NULL;

-- +goose Down
ALTER TABLE games DROP COLUMN details;
