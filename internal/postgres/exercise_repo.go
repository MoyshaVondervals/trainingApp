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
	const q = `INSERT INTO exercises (name, description, user_id) VALUES ($1, $2, $3) RETURNING id, created_at`
	err := r.db.QueryRowContext(ctx, q, e.Name, e.Description, e.UserID).Scan(&e.ID, &e.CreatedAt)
	if err != nil {
		return exercise.Exercise{}, fmt.Errorf("create exercise: %w", err)
	}
	return e, nil
}

func (r *ExerciseRepo) Delete(ctx context.Context, userID, id int64) error {
	const q = `DELETE FROM exercises WHERE id = $1 AND user_id = $2 RETURNING id`
	err := r.db.QueryRowContext(ctx, q, id, userID).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return exercise.ErrNotFound
		}
		return fmt.Errorf("delete exercise: %w", err)
	}
	return nil
}

func (r *ExerciseRepo) UpdateByID(ctx context.Context, userID int64, e exercise.Exercise) (exercise.Exercise, error) {
	const q = `UPDATE exercises SET name = $1, description = $2 WHERE id = $3 AND user_id = $4 RETURNING id, user_id, created_at`
	err := r.db.QueryRowContext(ctx, q, e.Name, e.Description, e.ID, userID).Scan(&e.ID, &e.UserID, &e.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return exercise.Exercise{}, exercise.ErrNotFound
		}
		return exercise.Exercise{}, fmt.Errorf("update exercise: %w", err)
	}
	return e, nil
}

func (r *ExerciseRepo) GetByID(ctx context.Context, userID, id int64) (exercise.Exercise, error) {
	const q = `SELECT id, name, description, user_id, created_at FROM exercises WHERE id = $1 AND (user_id IS NULL OR user_id = $2)`

	var e exercise.Exercise
	err := r.db.QueryRowContext(ctx, q, id, userID).Scan(&e.ID, &e.Name, &e.Description, &e.UserID, &e.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return exercise.Exercise{}, exercise.ErrNotFound
		}
		return exercise.Exercise{}, fmt.Errorf("get exercise %d: %w", id, err)
	}
	return e, nil
}

func (r *ExerciseRepo) List(ctx context.Context, userID int64, limit int) ([]exercise.Exercise, error) {
	const q = `SELECT id, name, description, user_id, created_at FROM exercises WHERE (user_id IS NULL OR user_id = $1) ORDER BY id LIMIT $2`
	rows, err := r.db.QueryContext(ctx, q, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("listing exercises: %w", err)
	}
	defer rows.Close()
	result := make([]exercise.Exercise, 0, limit)
	for rows.Next() {
		var e exercise.Exercise
		if err := rows.Scan(&e.ID, &e.Name, &e.Description, &e.UserID, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("listing exercises: %w", err)
		}
		result = append(result, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing exercises: %w", err)
	}
	return result, nil
}
