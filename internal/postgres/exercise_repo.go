package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"trainingApp/internal/exercise"

	"github.com/jmoiron/sqlx"
)

type ExerciseRepo struct{ db *sqlx.DB }

func NewExerciseRepo(db *sqlx.DB) *ExerciseRepo {
	return &ExerciseRepo{db: db}
}

func (r *ExerciseRepo) Create(ctx context.Context, e exercise.Exercise) (exercise.Exercise, error) {
	const q = `INSERT INTO exercises (name, description, user_id) VALUES ($1, $2, $3)
RETURNING id, name, description, user_id, created_at`
	var created exercise.Exercise
	err := r.db.GetContext(ctx, &created, q, e.Name, e.Description, e.UserID)
	if err != nil {
		if isUniqueViolation(err) {
			return exercise.Exercise{}, exercise.ErrAlreadyExists
		}
		return exercise.Exercise{}, fmt.Errorf("create exercise: %w", err)
	}
	return created, nil
}

func (r *ExerciseRepo) Delete(ctx context.Context, userID, id int64) error {
	const q = `DELETE FROM exercises WHERE id = $1 AND user_id = $2 RETURNING id`
	var deleted int64
	err := r.db.GetContext(ctx, &deleted, q, id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return exercise.ErrNotFound
		}
		return fmt.Errorf("delete exercise: %w", err)
	}
	return nil
}

func (r *ExerciseRepo) UpdateByID(ctx context.Context, userID int64, e exercise.Exercise) (exercise.Exercise, error) {
	const q = `UPDATE exercises SET name = $1, description = $2 WHERE id = $3 AND user_id = $4
RETURNING id, name, description, user_id, created_at`
	var updated exercise.Exercise
	err := r.db.GetContext(ctx, &updated, q, e.Name, e.Description, e.ID, userID)
	if err != nil {
		if isUniqueViolation(err) {
			return exercise.Exercise{}, exercise.ErrAlreadyExists
		}
		if errors.Is(err, sql.ErrNoRows) {
			return exercise.Exercise{}, exercise.ErrNotFound
		}
		return exercise.Exercise{}, fmt.Errorf("update exercise: %w", err)
	}
	return updated, nil
}

func (r *ExerciseRepo) GetByID(ctx context.Context, userID, id int64) (exercise.Exercise, error) {
	const q = `SELECT id, name, description, user_id, created_at FROM exercises
WHERE id = $1 AND (user_id IS NULL OR user_id = $2)`
	var e exercise.Exercise
	err := r.db.GetContext(ctx, &e, q, id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return exercise.Exercise{}, exercise.ErrNotFound
		}
		return exercise.Exercise{}, fmt.Errorf("get exercise %d: %w", id, err)
	}
	return e, nil
}

func (r *ExerciseRepo) List(ctx context.Context, userID int64, limit int) ([]exercise.Exercise, error) {
	const q = `SELECT id, name, description, user_id, created_at FROM exercises
WHERE (user_id IS NULL OR user_id = $1) ORDER BY id LIMIT $2`
	result := make([]exercise.Exercise, 0, limit)
	if err := r.db.SelectContext(ctx, &result, q, userID, limit); err != nil {
		return nil, fmt.Errorf("list exercises: %w", err)
	}
	return result, nil
}
