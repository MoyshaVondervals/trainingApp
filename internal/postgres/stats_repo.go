package postgres

import (
	"context"
	"fmt"
	"trainingApp/internal/stats"

	"github.com/jmoiron/sqlx"
)

type StatsRepo struct {
	db *sqlx.DB
}

func NewStatsRepo(db *sqlx.DB) *StatsRepo {
	return &StatsRepo{db: db}
}

func (r *StatsRepo) MuscleLoad(ctx context.Context, userID int64, p stats.Period) ([]stats.GroupRoleLoad, error) {
	const q = `SELECT mg.code,
       mg.name,
       reg.name AS region,
       em.role,
       SUM(s.reps * COALESCE(s.weight_kg, 0)) AS volume,
       SUM(s.reps) AS reps,
       COUNT(*) AS sets
FROM sets s
JOIN workouts w          ON w.id = s.workout_id
JOIN exercise_muscles em ON em.exercise_id = s.exercise_id
JOIN muscle_groups mg    ON mg.id = em.muscle_group_id
JOIN muscle_regions reg  ON reg.id = mg.parent_id
WHERE w.user_id = $1 AND w.started_at >= $2 AND w.started_at < $3
GROUP BY mg.code, mg.name, reg.name, em.role
ORDER BY volume DESC`
	res := make([]stats.GroupRoleLoad, 0, 16)
	if err := r.db.SelectContext(ctx, &res, q, userID, p.From, p.To); err != nil {
		return nil, fmt.Errorf("muscle load: %w", err)
	}
	return res, nil
}

func (r *StatsRepo) Records(ctx context.Context, userID int64, limit int) ([]stats.Record, error) {
	const q = `SELECT exercise_id, exercise_name, weight_kg, reps, achieved_at
FROM (
    SELECT e.id   AS exercise_id,
           e.name AS exercise_name,
           s.weight_kg,
           s.reps,
           w.started_at AS achieved_at,
           ROW_NUMBER() OVER (
               PARTITION BY e.id
               ORDER BY s.weight_kg DESC NULLS LAST, s.reps DESC
           ) AS rn
    FROM sets s
    JOIN workouts w  ON w.id = s.workout_id
    JOIN exercises e ON e.id = s.exercise_id
    WHERE w.user_id = $1
) ranked
WHERE rn = 1
ORDER BY weight_kg DESC NULLS LAST, reps DESC
LIMIT $2`
	res := make([]stats.Record, 0, limit)
	if err := r.db.SelectContext(ctx, &res, q, userID, limit); err != nil {
		return nil, fmt.Errorf("records: %w", err)
	}
	return res, nil
}

func (r *StatsRepo) Summary(ctx context.Context, userID int64, p stats.Period) (stats.Summary, error) {
	const q = `SELECT COUNT(DISTINCT w.id) AS workouts,
       COUNT(s.id) AS sets,
       COALESCE(SUM(s.reps), 0) AS reps,
       COALESCE(SUM(s.reps * COALESCE(s.weight_kg, 0)), 0) AS volume
FROM workouts w
LEFT JOIN sets s ON s.workout_id = w.id
WHERE w.user_id = $1 AND w.started_at >= $2 AND w.started_at < $3`
	var res stats.Summary
	if err := r.db.GetContext(ctx, &res, q, userID, p.From, p.To); err != nil {
		return stats.Summary{}, fmt.Errorf("summary: %w", err)
	}
	return res, nil
}
