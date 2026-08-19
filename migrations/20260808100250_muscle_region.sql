-- +goose Up
CREATE TABLE muscle_regions(
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    code VARCHAR(50) NOT NULL UNIQUE,
    sort_order int NOT NULL DEFAULT 0
);

INSERT INTO muscle_regions (code, name, sort_order) VALUES
                                                                    ('chest', 'Грудь',  10),
                                                                    ('back', 'Спина',  20),
                                                                    ('shoulders', 'Плечи',  30),
                                                                    ('arms', 'Руки',  40),
                                                                    ('core', 'Пресс и кор',  50),
                                                                    ('legs', 'Ноги',  60),
                                                                    ('neck', 'Шея',  70)
ON CONFLICT (code) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS muscle_regions CASCADE;