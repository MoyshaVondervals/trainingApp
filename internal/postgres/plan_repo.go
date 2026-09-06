package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"trainingApp/internal/plan"

	"github.com/jmoiron/sqlx"
)

const planColumns = `id, user_id, name, note, created_at`

type PlanRepo struct {
	db *sqlx.DB
}

func NewPlanRepo(db *sqlx.DB) *PlanRepo {
	return &PlanRepo{db: db}
}

func (r *PlanRepo) Create(ctx context.Context, p plan.Plan) (plan.Plan, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	const q = `INSERT INTO plans (user_id, name, note) VALUES ($1, $2, $3) RETURNING ` + planColumns
	var created plan.Plan
	if err := tx.GetContext(ctx, &created, q, p.UserID, p.Name, p.Note); err != nil {
		if isUniqueViolation(err) {
			return plan.Plan{}, plan.ErrAlreadyExists
		}
		return plan.Plan{}, fmt.Errorf("create plan: %w", err)
	}

	if err := insertPlanItems(ctx, tx, created.ID, p.UserID, p.Exercises); err != nil {
		return plan.Plan{}, err
	}
	if err := tx.Commit(); err != nil {
		return plan.Plan{}, fmt.Errorf("commit: %w", err)
	}

	created.Exercises = p.Exercises
	return created, nil
}

func (r *PlanRepo) Update(ctx context.Context, userID int64, p plan.Plan) (plan.Plan, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	const updateQ = `UPDATE plans SET name = $1, note = $2 WHERE id = $3 AND user_id = $4 RETURNING ` + planColumns
	var updated plan.Plan
	if err := tx.GetContext(ctx, &updated, updateQ, p.Name, p.Note, p.ID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return plan.Plan{}, plan.ErrNotFound
		}
		if isUniqueViolation(err) {
			return plan.Plan{}, plan.ErrAlreadyExists
		}
		return plan.Plan{}, fmt.Errorf("update plan %d: %w", p.ID, err)
	}

	const deleteQ = `DELETE FROM plan_exercises WHERE plan_id = $1`
	if _, err := tx.ExecContext(ctx, deleteQ, p.ID); err != nil {
		return plan.Plan{}, fmt.Errorf("clear plan_exercises: %w", err)
	}
	if err := insertPlanItems(ctx, tx, p.ID, userID, p.Exercises); err != nil {
		return plan.Plan{}, err
	}
	if err := tx.Commit(); err != nil {
		return plan.Plan{}, fmt.Errorf("commit: %w", err)
	}

	updated.Exercises = p.Exercises
	return updated, nil
}

func insertPlanItems(ctx context.Context, tx *sqlx.Tx, planID, userID int64, items []plan.Item) error {
	const q = `INSERT INTO plan_exercises (plan_id, exercise_id, position, target_sets, target_reps) 
		SELECT $1, e.id, $3, $4, $5 FROM exercises e WHERE e.id = $2 AND (e.user_id IS NULL OR e.user_id = $6)`

	for _, item := range items {
		res, err := tx.ExecContext(ctx, q,
			planID, item.ExerciseID, item.Position, item.TargetSets, item.TargetReps, userID)
		if err != nil {
			return fmt.Errorf("insert plan exercise %d: %w", item.ExerciseID, err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("insert plan exercise %d: %w", item.ExerciseID, err)
		}
		if affected == 0 {
			return plan.ErrNotFound
		}
	}
	return nil
}

func (r *PlanRepo) List(ctx context.Context, userID int64, limit int) ([]plan.Plan, error) {
	const q = `SELECT ` + planColumns + ` FROM plans WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`
	plans := make([]plan.Plan, 0, limit)
	if err := r.db.SelectContext(ctx, &plans, q, userID, limit); err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	return plans, nil
}

func (r *PlanRepo) GetByID(ctx context.Context, userID, id int64) (plan.Plan, error) {
	const planQ = `SELECT ` + planColumns + ` FROM plans WHERE id = $1 AND user_id = $2`
	var p plan.Plan
	if err := r.db.GetContext(ctx, &p, planQ, id, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return plan.Plan{}, plan.ErrNotFound
		}
		return plan.Plan{}, fmt.Errorf("get plan %d: %w", id, err)
	}

	items, err := r.items(ctx, id)
	if err != nil {
		return plan.Plan{}, err
	}
	p.Exercises = items
	return p, nil
}

func (r *PlanRepo) items(ctx context.Context, planID int64) ([]plan.Item, error) {
	const q = `SELECT pe.exercise_id, e.name AS exercise_name, pe.position, pe.target_sets, pe.target_reps
		FROM plan_exercises pe 
		JOIN exercises e ON e.id = pe.exercise_id
		WHERE pe.plan_id = $1
		ORDER BY pe.position`
	items := make([]plan.Item, 0, 8)
	if err := r.db.SelectContext(ctx, &items, q, planID); err != nil {
		return nil, fmt.Errorf("list plan exercises: %w", err)
	}
	return items, nil
}

func (r *PlanRepo) Delete(ctx context.Context, userID, id int64) error {
	const q = `DELETE FROM plans WHERE id = $1 AND user_id = $2`
	res, err := r.db.ExecContext(ctx, q, id, userID)
	if err != nil {
		return fmt.Errorf("delete plan %d: %w", id, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete plan %d: %w", id, err)
	}
	if affected == 0 {
		return plan.ErrNotFound
	}
	return nil
}
