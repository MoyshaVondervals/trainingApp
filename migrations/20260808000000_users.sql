-- +goose Up

CREATE EXTENSION IF NOT EXISTS citext;
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    second_name VARCHAR(100) NOT NULL,
    email CITEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
drop table if exists users cascade;
DROP EXTENSION IF EXISTS citext;