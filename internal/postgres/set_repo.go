package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"trainingApp/internal/set"
)

type SetRepo struct {
	db *sql.DB
}

func NewSetRepo(db *sql.DB) *SetRepo {
	return &SetRepo{db: db}
}

func (r *SetRepo) Create(ctx context.Context, userID int64, s set.Set) (set.Set, error) {
	const q = `INSERT INTO sets (exercise_id, workout_id, set_number, reps, weight_kg)
SELECT $1, w.id, $3, $4, $5
FROM workouts w
WHERE w.id = $2 AND w.user_id = $6
RETURNING id, created_at`
	err := r.db.QueryRowContext(ctx, q, s.ExerciseID, s.WorkoutID, s.SetNumber, s.Reps, s.Weight, userID).
		Scan(&s.ID, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return set.Set{}, set.ErrNotFound
		}
		return set.Set{}, fmt.Errorf("create set: %w", err)
	}
	return s, nil
}

func (r *SetRepo) Update(ctx context.Context, userID int64, s set.Set) (set.Set, error) {
	const q = `UPDATE sets s SET set_number = $1, reps = $2, weight_kg = $3
FROM workouts w
WHERE s.workout_id = w.id AND s.id = $4 AND w.user_id = $5
RETURNING s.id, s.exercise_id, s.workout_id, s.set_number, s.reps, s.weight_kg, s.created_at`
	var e set.Set
	err := r.db.QueryRowContext(ctx, q, s.SetNumber, s.Reps, s.Weight, s.ID, userID).
		Scan(&e.ID, &e.ExerciseID, &e.WorkoutID, &e.SetNumber, &e.Reps, &e.Weight, &e.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return set.Set{}, set.ErrNotFound
		}
		return set.Set{}, fmt.Errorf("update set: %w", err)
	}
	return e, nil
}

func (r *SetRepo) Delete(ctx context.Context, userID int64, s set.Set) error {
	const q = `DELETE FROM sets s USING workouts w WHERE s.workout_id = w.id AND s.id = $1 AND w.user_id = $2 RETURNING s.id`
	err := r.db.QueryRowContext(ctx, q, s.ID, userID).Scan(&s.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return set.ErrNotFound
		}
		return fmt.Errorf("delete set: %w", err)
	}
	return nil
}

func (r *SetRepo) GetById(ctx context.Context, userID, id int64) (set.Set, error) {
	const q = `SELECT s.id, s.exercise_id, s.workout_id, s.set_number, s.reps, s.weight_kg, s.created_at
FROM sets s
JOIN workouts w ON w.id = s.workout_id
WHERE s.id = $1 AND w.user_id = $2`
	var s set.Set
	err := r.db.QueryRowContext(ctx, q, id, userID).Scan(&s.ID, &s.ExerciseID, &s.WorkoutID, &s.SetNumber, &s.Reps, &s.Weight, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return set.Set{}, set.ErrNotFound
		}
		return set.Set{}, fmt.Errorf("get set by id: %w", err)
	}
	return s, nil
}

func (r *SetRepo) ListByWorkout(ctx context.Context, userID, workoutID int64, limit int) ([]set.Set, error) {
	const q = `SELECT s.id, s.exercise_id, s.workout_id, s.set_number, s.reps, s.weight_kg, s.created_at
FROM sets s
JOIN workouts w ON w.id = s.workout_id
WHERE s.workout_id = $1 AND w.user_id = $2
ORDER BY s.set_number
LIMIT $3`
	rows, err := r.db.QueryContext(ctx, q, workoutID, userID, limit)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []set.Set{}, set.ErrNotFound
		}
		return []set.Set{}, fmt.Errorf("get sets by workout: %w", err)
	}
	defer rows.Close()
	res := make([]set.Set, 0, limit)
	for rows.Next() {
		var s set.Set
		if err := rows.Scan(&s.ID, &s.ExerciseID, &s.WorkoutID, &s.SetNumber, &s.Reps, &s.Weight, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("get sets by workout: %w", err)
		}
		res = append(res, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get sets by workout: %w", err)
	}
	return res, nil
}
