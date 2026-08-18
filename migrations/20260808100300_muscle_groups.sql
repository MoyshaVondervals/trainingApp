-- +goose Up
create table muscle_groups(
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL UNIQUE
);

-- +goose Down
drop table if exists muscle_groups cascade;
