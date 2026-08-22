package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"trainingApp/internal/workout"

	"github.com/jmoiron/sqlx"
)

const workoutColumns = `id, user_id, started_at, ended_at, note`

type WorkoutRepo struct {
	db *sqlx.DB
}

func NewWorkoutRepo(db *sqlx.DB) *WorkoutRepo {
	return &WorkoutRepo{db: db}
}

func (r *WorkoutRepo) Create(ctx context.Context, w workout.Workout) (workout.Workout, error) {
	const q = `INSERT INTO workouts (user_id, started_at, note) VALUES ($1, $2, $3)
RETURNING ` + workoutColumns
	var created workout.Workout
	err := r.db.GetContext(ctx, &created, q, w.UserID, w.StartedAt, w.Note)
	if err != nil {
		return workout.Workout{}, fmt.Errorf("create workout: %w", err)
	}
	return created, nil
}

func (r *WorkoutRepo) GetByID(ctx context.Context, userID, id int64) (workout.Workout, error) {
	const q = `SELECT ` + workoutColumns + ` FROM workouts WHERE id = $1 AND user_id = $2`
	var w workout.Workout
	err := r.db.GetContext(ctx, &w, q, id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return workout.Workout{}, workout.ErrNotFound
		}
		return workout.Workout{}, fmt.Errorf("get workout %d: %w", id, err)
	}
	return w, nil
}

func (r *WorkoutRepo) List(ctx context.Context, userID int64, limit int) ([]workout.Workout, error) {
	const q = `SELECT ` + workoutColumns + ` FROM workouts WHERE user_id = $1
ORDER BY started_at DESC LIMIT $2`
	res := make([]workout.Workout, 0, limit)
	if err := r.db.SelectContext(ctx, &res, q, userID, limit); err != nil {
		return nil, fmt.Errorf("list workouts: %w", err)
	}
	return res, nil
}

func (r *WorkoutRepo) UpdateNoteByID(ctx context.Context, userID int64, e workout.Workout) (workout.Workout, error) {
	const q = `UPDATE workouts SET note = $1 WHERE id = $2 AND user_id = $3
RETURNING ` + workoutColumns
	var w workout.Workout
	err := r.db.GetContext(ctx, &w, q, e.Note, e.ID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return workout.Workout{}, workout.ErrNotFound
		}
		return workout.Workout{}, fmt.Errorf("update workout note %d: %w", e.ID, err)
	}
	return w, nil
}

func (r *WorkoutRepo) FinishTraining(ctx context.Context, userID, id int64) (workout.Workout, error) {
	const q = `UPDATE workouts SET ended_at = now()
WHERE id = $1 AND user_id = $2 AND ended_at IS NULL
RETURNING ` + workoutColumns
	var w workout.Workout
	err := r.db.GetContext(ctx, &w, q, id, userID)
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
	var deleted int64
	err := r.db.GetContext(ctx, &deleted, q, id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return workout.ErrNotFound
		}
		return fmt.Errorf("delete workout: %w", err)
	}
	return nil
}
