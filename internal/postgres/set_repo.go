package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"trainingApp/internal/set"

	"github.com/jmoiron/sqlx"
)

const setColumns = `s.id, s.exercise_id, s.workout_id, s.set_number, s.reps, s.weight_kg, s.created_at`

type SetRepo struct {
	db *sqlx.DB
}

func NewSetRepo(db *sqlx.DB) *SetRepo {
	return &SetRepo{db: db}
}

func (r *SetRepo) Create(ctx context.Context, userID int64, s set.Set) (set.Set, error) {
	const q = `INSERT INTO sets (exercise_id, workout_id, set_number, reps, weight_kg)
SELECT $1, w.id, $3, $4, $5
FROM workouts w
WHERE w.id = $2 AND w.user_id = $6
RETURNING id, exercise_id, workout_id, set_number, reps, weight_kg, created_at`
	var created set.Set
	err := r.db.GetContext(ctx, &created, q, s.ExerciseID, s.WorkoutID, s.SetNumber, s.Reps, s.Weight, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return set.Set{}, set.ErrNotFound
		}
		return set.Set{}, fmt.Errorf("create set: %w", err)
	}
	return created, nil
}

func (r *SetRepo) Update(ctx context.Context, userID int64, s set.Set) (set.Set, error) {
	const q = `UPDATE sets s SET set_number = $1, reps = $2, weight_kg = $3
FROM workouts w
WHERE s.workout_id = w.id AND s.id = $4 AND w.user_id = $5
RETURNING ` + setColumns
	var updated set.Set
	err := r.db.GetContext(ctx, &updated, q, s.SetNumber, s.Reps, s.Weight, s.ID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return set.Set{}, set.ErrNotFound
		}
		return set.Set{}, fmt.Errorf("update set: %w", err)
	}
	return updated, nil
}

func (r *SetRepo) Delete(ctx context.Context, userID int64, s set.Set) error {
	const q = `DELETE FROM sets s
USING workouts w
WHERE s.workout_id = w.id AND s.id = $1 AND w.user_id = $2
RETURNING s.id`
	var deleted int64
	err := r.db.GetContext(ctx, &deleted, q, s.ID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return set.ErrNotFound
		}
		return fmt.Errorf("delete set: %w", err)
	}
	return nil
}

func (r *SetRepo) GetById(ctx context.Context, userID, id int64) (set.Set, error) {
	const q = `SELECT ` + setColumns + `
FROM sets s
JOIN workouts w ON w.id = s.workout_id
WHERE s.id = $1 AND w.user_id = $2`
	var s set.Set
	err := r.db.GetContext(ctx, &s, q, id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return set.Set{}, set.ErrNotFound
		}
		return set.Set{}, fmt.Errorf("get set %d: %w", id, err)
	}
	return s, nil
}

func (r *SetRepo) ListByWorkout(ctx context.Context, userID, workoutID int64, limit int) ([]set.Set, error) {
	const q = `SELECT ` + setColumns + `
FROM sets s
JOIN workouts w ON w.id = s.workout_id
WHERE s.workout_id = $1 AND w.user_id = $2
ORDER BY s.set_number
LIMIT $3`
	res := make([]set.Set, 0, limit)
	if err := r.db.SelectContext(ctx, &res, q, workoutID, userID, limit); err != nil {
		return nil, fmt.Errorf("list sets: %w", err)
	}
	return res, nil
}
