package postgres

import (
	"context"
	"fmt"
	"time"
	"trainingApp/internal/weight"

	"github.com/jmoiron/sqlx"
)

const bodyWeightColumns = `id, user_id, weight_kg, measured_on, note, created_at`

type WeightRepo struct {
	db *sqlx.DB
}

func NewWeightRepo(db *sqlx.DB) *WeightRepo {
	return &WeightRepo{db: db}
}

func (r *WeightRepo) Upsert(ctx context.Context, b weight.BodyWeight) (weight.BodyWeight, error) {
	const q = `INSERT INTO body_weights (user_id, weight_kg, measured_on, note)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, measured_on) DO UPDATE SET weight_kg = EXCLUDED.weight_kg, note = EXCLUDED.note
RETURNING ` + bodyWeightColumns
	var created weight.BodyWeight
	err := r.db.GetContext(ctx, &created, q, b.UserID, b.WeightKg, b.MeasuredOn, b.Note)
	if err != nil {
		return weight.BodyWeight{}, fmt.Errorf("upsert body weight: %w", err)
	}
	return created, nil
}

func (r *WeightRepo) List(ctx context.Context, userID int64, from, to time.Time, limit int) ([]weight.BodyWeight, error) {
	const q = `SELECT ` + bodyWeightColumns + ` FROM body_weights WHERE user_id = $1 AND measured_on >= $2 AND measured_on <= $3 ORDER BY measured_on DESC LIMIT $4`
	res := make([]weight.BodyWeight, 0, limit)
	if err := r.db.SelectContext(ctx, &res, q, userID, from, to, limit); err != nil {
		return nil, fmt.Errorf("list body weights: %w", err)
	}
	return res, nil
}

func (r *WeightRepo) Delete(ctx context.Context, userID, id int64) error {
	const q = `DELETE FROM body_weights WHERE id = $1 AND user_id = $2`
	res, err := r.db.ExecContext(ctx, q, id, userID)
	if err != nil {
		return fmt.Errorf("delete body weight %d: %w", id, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete body weight %d: %w", id, err)
	}
	if affected == 0 {
		return weight.ErrNotFound
	}
	return nil
}
