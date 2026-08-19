-- +goose Up
create table sets(
    id BIGSERIAL PRIMARY KEY,
    exercise_id BIGINT NOT NULL REFERENCES exercises(id) ON DELETE RESTRICT,
    workout_id BIGINT NOT NULL REFERENCES workouts(id) ON DELETE CASCADE,
    set_number INT NOT NULL CHECK ( set_number>0 ),
    reps INT NOT NULL CHECK ( reps >0 ),
    weight_kg NUMERIC(6,2) CHECK ( weight_kg is NULL OR weight_kg>=0),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (workout_id, exercise_id, set_number)
);

CREATE INDEX sets_workout_idx ON sets(workout_id);
CREATE INDEX sets_exercise_idx ON sets(exercise_id);
-- +goose Down
drop table if exists sets cascade;