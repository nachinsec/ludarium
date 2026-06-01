-- +goose Up
ALTER TABLE users ADD COLUMN ai_base_url TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN ai_model    TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN ai_api_key  TEXT NOT NULL DEFAULT ''; -- AES-GCM encrypted

-- +goose Down
ALTER TABLE users DROP COLUMN ai_base_url;
ALTER TABLE users DROP COLUMN ai_model;
ALTER TABLE users DROP COLUMN ai_api_key;
