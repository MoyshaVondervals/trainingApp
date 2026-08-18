
-- +goose Up
create table workouts(
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at TIMESTAMPTZ DEFAULT NULL CHECK (ended_at IS NULL OR ended_at >= started_at),
    note VARCHAR(1000) NOT NULL DEFAULT ''
);


-- +goose Down
drop table if exists workouts cascade;