
-- +goose Up
create table exercises(
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(1000) NOT NULL DEFAULT '',
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX exercises_name_system_idx ON exercises (lower(name)) WHERE user_id IS NULL;
CREATE UNIQUE INDEX exercises_name_user_idx   ON exercises (user_id, lower(name)) WHERE user_id IS NOT NULL;
CREATE INDEX exercises_user_idx ON exercises (user_id);

-- +goose Down
drop table if exists exercises cascade;