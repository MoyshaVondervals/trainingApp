-- +goose Up
create table exercise_muscles(
    exercise_id BIGINT NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
    muscule_group_id BIGINT NOT NULL REFERENCES muscle_groups(id) ON DELETE RESTRICT,
    role VARCHAR(20) NOT NULL CHECK ( role IN ('primary', 'secondary') ),
    technic_description VARCHAR (500) NOT NULL,
    PRIMARY KEY (exercise_id, muscule_group_id)
);
CREATE INDEX exercise_muscles_muscle_idx ON exercise_muscles(muscule_group_id);


-- +goose Down
drop table if exists exercise_muscles cascade;