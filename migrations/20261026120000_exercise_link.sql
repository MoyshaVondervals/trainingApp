-- +goose Up
ALTER TABLE exercises ADD COLUMN link VARCHAR(500) NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE exercises DROP COLUMN IF EXISTS link;
