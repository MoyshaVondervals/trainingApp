
-- +goose Up
create table workouts(
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at TIMESTAMPTZ DEFAULT NULL CHECK (ended_at IS NULL OR ended_at >= started_at),
    note VARCHAR(1000) NOT NULL DEFAULT ''
);
CREATE INDEX workouts_user_started_idx ON workouts(user_id, started_at DESC );


-- +goose Down
drop table if exists workouts cascade;