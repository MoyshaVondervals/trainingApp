package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"trainingApp/internal/exercise"

	"github.com/jackc/pgx/v5/pgconn"
)

type ExerciseMusclesRepo struct {
	db *sql.DB
}

func NewExerciseMusclesRepo(db *sql.DB) *ExerciseMusclesRepo {
	return &ExerciseMusclesRepo{db: db}
}

func (r *ExerciseMusclesRepo) ListByExercise(ctx context.Context, userID, exerciseID int64) ([]exercise.Muscle, error) {
	const q = `SELECT em.muscle_group_id, em.role
FROM exercise_muscles em
JOIN exercises e ON e.id = em.exercise_id
WHERE em.exercise_id = $1 AND (e.user_id IS NULL OR e.user_id = $2)
ORDER BY em.role, em.muscle_group_id`
	rows, err := r.db.QueryContext(ctx, q, exerciseID, userID)
	if err != nil {
		return nil, fmt.Errorf("select exercise_muscles: %w", err)
	}
	defer rows.Close()

	muscles := make([]exercise.Muscle, 0, 8)
	for rows.Next() {
		var m exercise.Muscle
		if err := rows.Scan(&m.MuscleGroupID, &m.Role); err != nil {
			return nil, fmt.Errorf("scan exercise_muscles: %w", err)
		}
		muscles = append(muscles, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("select exercise_muscles: %w", err)
	}
	return muscles, nil
}

func (r *ExerciseMusclesRepo) ReplaceForExercise(ctx context.Context, userID, exerciseID int64, muscles []exercise.Muscle) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	const ownerQ = `SELECT id FROM exercises WHERE id = $1 AND user_id = $2`
	var id int64
	if err := tx.QueryRowContext(ctx, ownerQ, exerciseID, userID).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return exercise.ErrNotFound
		}
		return fmt.Errorf("check exercise owner: %w", err)
	}

	const deleteQ = `DELETE FROM exercise_muscles WHERE exercise_id = $1`
	if _, err := tx.ExecContext(ctx, deleteQ, exerciseID); err != nil {
		return fmt.Errorf("clear exercise_muscles: %w", err)
	}

	const insertQ = `INSERT INTO exercise_muscles (exercise_id, muscle_group_id, role) VALUES ($1, $2, $3)`
	stmt, err := tx.PrepareContext(ctx, insertQ)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, m := range muscles {
		if _, err := stmt.ExecContext(ctx, exerciseID, m.MuscleGroupID, m.Role); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				return exercise.ErrMuscleGroupNotFound
			}
			return fmt.Errorf("insert muscle %d: %w", m.MuscleGroupID, err)
		}
	}

	return tx.Commit()
}
