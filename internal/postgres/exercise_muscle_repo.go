package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"trainingApp/internal/exercise"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

type ExerciseMusclesRepo struct {
	db *sqlx.DB
}

func NewExerciseMusclesRepo(db *sqlx.DB) *ExerciseMusclesRepo {
	return &ExerciseMusclesRepo{db: db}
}

func (r *ExerciseMusclesRepo) ListByExercise(ctx context.Context, userID, exerciseID int64) ([]exercise.Muscle, error) {
	const q = `SELECT em.muscle_group_id, em.role
FROM exercise_muscles em
JOIN exercises e ON e.id = em.exercise_id
WHERE em.exercise_id = $1 AND (e.user_id IS NULL OR e.user_id = $2)
ORDER BY em.role, em.muscle_group_id`
	muscles := make([]exercise.Muscle, 0, 8)
	if err := r.db.SelectContext(ctx, &muscles, q, exerciseID, userID); err != nil {
		return nil, fmt.Errorf("select exercise_muscles: %w", err)
	}
	return muscles, nil
}

func (r *ExerciseMusclesRepo) ReplaceForExercise(ctx context.Context, userID, exerciseID int64, muscles []exercise.Muscle) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	const ownerQ = `SELECT id FROM exercises WHERE id = $1 AND user_id = $2`
	var id int64
	if err := tx.GetContext(ctx, &id, ownerQ, exerciseID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return exercise.ErrNotFound
		}
		return fmt.Errorf("check exercise owner: %w", err)
	}

	const deleteQ = `DELETE FROM exercise_muscles WHERE exercise_id = $1`
	if _, err := tx.ExecContext(ctx, deleteQ, exerciseID); err != nil {
		return fmt.Errorf("clear exercise_muscles: %w", err)
	}

	const insertQ = `INSERT INTO exercise_muscles (exercise_id, muscle_group_id, role)
VALUES (:exercise_id, :muscle_group_id, :role)`
	type row struct {
		ExerciseID    int64  `db:"exercise_id"`
		MuscleGroupID int64  `db:"muscle_group_id"`
		Role          string `db:"role"`
	}
	rows := make([]row, 0, len(muscles))
	for _, m := range muscles {
		rows = append(rows, row{ExerciseID: exerciseID, MuscleGroupID: m.MuscleGroupID, Role: m.Role})
	}
	if _, err := tx.NamedExecContext(ctx, insertQ, rows); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return exercise.ErrMuscleGroupNotFound
		}
		return fmt.Errorf("insert exercise_muscles: %w", err)
	}

	return tx.Commit()
}
