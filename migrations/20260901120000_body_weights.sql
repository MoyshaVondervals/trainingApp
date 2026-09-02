-- +goose Up
CREATE TABLE body_weights(
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    weight_kg NUMERIC(5,2) NOT NULL CHECK ( weight_kg > 0 AND weight_kg <= 500 ),
    measured_on DATE NOT NULL,
    note VARCHAR(200) NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, measured_on)
);

CREATE INDEX body_weights_user_measured_idx ON body_weights(user_id, measured_on DESC);

-- +goose Down
DROP TABLE IF EXISTS body_weights CASCADE;
