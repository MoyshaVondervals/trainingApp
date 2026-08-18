-- +goose Up
create table sets(
    id BIGSERIAL PRIMARY KEY,
    exercise_id BIGINT NOT NULL REFERENCES exercises(id) ON DELETE RESTRICT,
    workouts_id BIGINT NOT NULL REFERENCES workouts(id) ON DELETE CASCADE,
    set_number INT NOT NULL,
    repeats INT NOT NULL,
    weight_kg NUMERIC(6,2) ,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
drop table if exists sets cascade;