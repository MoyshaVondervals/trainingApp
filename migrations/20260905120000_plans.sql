-- +goose Up
CREATE TABLE plans(
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    note VARCHAR(1000) NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, name)
);

CREATE TABLE plan_exercises(
    plan_id BIGINT NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    exercise_id BIGINT NOT NULL REFERENCES exercises(id) ON DELETE RESTRICT,
    position INT NOT NULL CHECK ( position > 0 ),
    target_sets INT NOT NULL CHECK ( target_sets > 0 AND target_sets <= 30 ),
    target_reps INT NOT NULL CHECK ( target_reps > 0 AND target_reps <= 5000 ),
    PRIMARY KEY (plan_id, exercise_id),
    UNIQUE (plan_id, position)
);

CREATE INDEX plan_exercises_exercise_idx ON plan_exercises(exercise_id);

ALTER TABLE workouts ADD COLUMN plan_id BIGINT REFERENCES plans(id) ON DELETE SET NULL;
CREATE INDEX workouts_plan_idx ON workouts(user_id, plan_id, started_at DESC);

-- +goose Down
DROP INDEX IF EXISTS workouts_plan_idx;
ALTER TABLE workouts DROP COLUMN IF EXISTS plan_id;
DROP TABLE IF EXISTS plan_exercises CASCADE;
DROP TABLE IF EXISTS plans CASCADE;
