package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"trainingApp/internal/exercise"
)

type ExerciseRepo struct{ db *sql.DB }

func NewExerciseRepo(db *sql.DB) *ExerciseRepo {
	return &ExerciseRepo{db: db}
}

func (r *ExerciseRepo) Create(ctx context.Context, e exercise.Exercise) (exercise.Exercise, error) {
	const q = `INSERT INTO exercises (name, description) VALUES ($1, $2) RETURNING id, created_at`
	err := r.db.QueryRowContext(ctx, q, e.Name, e.Description).Scan(&e.ID, &e.CreatedAt)
	if err != nil {
		return exercise.Exercise{}, fmt.Errorf("create exercise: %w", err)
	}
	return e, nil
}

func (r *ExerciseRepo) GetByID(ctx context.Context, id int64) (exercise.Exercise, error) {
	const q = `SELECT id, name, description, created_at FROM exercises WHERE id = $1`

	var e exercise.Exercise
	err := r.db.QueryRowContext(ctx, q, id).Scan(&e.ID, &e.Name, &e.Description, &e.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return exercise.Exercise{}, exercise.ErrNotFound
		}
		return exercise.Exercise{}, fmt.Errorf("get exercise %d: %w", id, err)
	}
	return e, nil
}

func (r *ExerciseRepo) List(ctx context.Context, limit int) ([]exercise.Exercise, error) {
	const q = `SELECT id, name, description, created_at FROM exercises ORDER BY id LIMIT $1`
	rows, err := r.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("listing exercises: %w", err)
	}
	defer rows.Close()
	result := make([]exercise.Exercise, 0, limit)
	for rows.Next() {
		var e exercise.Exercise
		if err := rows.Scan(&e.ID, &e.Name, &e.Description, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("listing exercises: %w", err)
		}
		result = append(result, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing exercises: %w", err)
	}
	return result, nil
}
