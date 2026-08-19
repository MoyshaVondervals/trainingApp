-- +goose Up
create table exercise_muscles(
    exercise_id BIGINT NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
    muscle_group_id BIGINT NOT NULL REFERENCES muscle_groups(id) ON DELETE RESTRICT,
    role VARCHAR(20) NOT NULL CHECK ( role IN ('primary', 'secondary') ),
    PRIMARY KEY (exercise_id, muscle_group_id)
);
CREATE INDEX exercise_muscles_muscle_idx ON exercise_muscles(muscle_group_id);


-- +goose Down
drop table if exists exercise_muscles cascade;