package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"trainingApp/internal/workout"
)

type WorkoutRepo struct {
	db *sql.DB
}

func NewWorkoutRepo(db *sql.DB) *WorkoutRepo {
	return &WorkoutRepo{db: db}
}

func (r *WorkoutRepo) Create(ctx context.Context, w workout.Workout) (workout.Workout, error) {
	const q = `INSERT INTO workouts (user_id, started_at, note) VALUES ($1, $2, $3) RETURNING id, started_at, ended_at`
	err := r.db.QueryRowContext(ctx, q, w.UserID, w.StartedAt, w.Note).Scan(&w.ID, &w.StartedAt, &w.EndedAt)
	if err != nil {
		return workout.Workout{}, fmt.Errorf("create workout: %w", err)
	}
	return w, nil
}

func (r *WorkoutRepo) GetByID(ctx context.Context, userID, id int64) (workout.Workout, error) {
	const q = `SELECT id, user_id, started_at, ended_at, note FROM workouts WHERE id = $1 AND user_id = $2`
	var w workout.Workout
	err := r.db.QueryRowContext(ctx, q, id, userID).Scan(&w.ID, &w.UserID, &w.StartedAt, &w.EndedAt, &w.Note)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return workout.Workout{}, workout.ErrNotFound
		}
		return workout.Workout{}, fmt.Errorf("get workout %d: %w", id, err)
	}
	return w, nil
}

func (r *WorkoutRepo) List(ctx context.Context, userID int64, limit int) ([]workout.Workout, error) {
	const q = `SELECT id, user_id, started_at, ended_at, note FROM workouts WHERE user_id = $1 ORDER BY started_at DESC LIMIT $2`
	rows, err := r.db.QueryContext(ctx, q, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list workouts: %w", err)
	}
	defer rows.Close()
	res := make([]workout.Workout, 0, limit)
	for rows.Next() {
		var w workout.Workout
		if err := rows.Scan(&w.ID, &w.UserID, &w.StartedAt, &w.EndedAt, &w.Note); err != nil {
			return nil, fmt.Errorf("list workouts: %w", err)
		}
		res = append(res, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list workouts: %w", err)
	}
	return res, nil
}

func (r *WorkoutRepo) UpdateNoteByID(ctx context.Context, userID int64, e workout.Workout) (workout.Workout, error) {
	const q = `UPDATE workouts SET note = $1 WHERE id = $2 AND user_id = $3 RETURNING id, user_id, started_at, ended_at, note`
	var w workout.Workout
	err := r.db.QueryRowContext(ctx, q, e.Note, e.ID, userID).
		Scan(&w.ID, &w.UserID, &w.StartedAt, &w.EndedAt, &w.Note)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return workout.Workout{}, workout.ErrNotFound
		}
		return workout.Workout{}, fmt.Errorf("update workout note %d: %w", e.ID, err)
	}
	return w, nil
}

func (r *WorkoutRepo) FinishTraining(ctx context.Context, userID, id int64) (workout.Workout, error) {
	const q = `UPDATE workouts SET ended_at = now() WHERE id = $1 AND user_id = $2 AND ended_at IS NULL RETURNING id, user_id, started_at, ended_at, note`
	var w workout.Workout
	err := r.db.QueryRowContext(ctx, q, id, userID).
		Scan(&w.ID, &w.UserID, &w.StartedAt, &w.EndedAt, &w.Note)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return workout.Workout{}, workout.ErrNotFound
		}
		return workout.Workout{}, fmt.Errorf("finish workout %d: %w", id, err)
	}
	return w, nil
}

func (r *WorkoutRepo) Delete(ctx context.Context, userID, id int64) error {
	const q = `DELETE FROM workouts WHERE id = $1 AND user_id = $2 RETURNING id`
	err := r.db.QueryRowContext(ctx, q, id, userID).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return workout.ErrNotFound
		}
		return fmt.Errorf("delete workout: %w", err)
	}
	return nil
}
