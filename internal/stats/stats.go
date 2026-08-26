package stats

import (
	"errors"
	"sort"
	"time"
)

const (
	RolePrimary = "primary"

	primaryWeight   = 1.0
	secondaryWeight = 0.5
	maxRecords      = 20
	defaultPeriod   = 90 * 24 * time.Hour
)

var ErrBadPeriod = errors.New("from must be earlier than to")

type MuscleLoad struct {
	Code   string  `json:"code"`
	Name   string  `json:"name"`
	Region string  `json:"region"`
	Volume float64 `json:"volume"`
	Reps   int64   `json:"reps"`
	Sets   int64   `json:"sets"`
}

type GroupRoleLoad struct {
	Code   string
	Name   string
	Region string
	Role   string
	Volume float64
	Reps   int64
	Sets   int64
}

type Record struct {
	ExerciseID   int64     `json:"exercise_id" db:"exercise_id"`
	ExerciseName string    `json:"exercise_name" db:"exercise_name"`
	WeightKg     *float64  `json:"weight_kg" db:"weight_kg"`
	Reps         int       `json:"reps"`
	AchievedAt   time.Time `json:"achieved_at" db:"achieved_at"`
}

type Summary struct {
	Workouts int64   `json:"workouts"`
	Sets     int64   `json:"sets"`
	Reps     int64   `json:"reps"`
	Volume   float64 `json:"volume"`
}

type Period struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type Dashboard struct {
	Period  Period       `json:"period"`
	Summary Summary      `json:"summary"`
	Muscles []MuscleLoad `json:"muscles"`
	Records []Record     `json:"records"`
}

func NewPeriod(from, to time.Time) (Period, error) {
	if to.IsZero() {
		to = time.Now()
	}
	if from.IsZero() {
		from = to.Add(-defaultPeriod)
	}
	if !from.Before(to) {
		return Period{}, ErrBadPeriod
	}
	return Period{From: from, To: to}, nil
}

func MaxRecords() int { return maxRecords }

func roleWeight(role string) float64 {
	if role == RolePrimary {
		return primaryWeight
	}
	return secondaryWeight
}

func AggregateByGroup(rows []GroupRoleLoad) []MuscleLoad {
	order := make([]string, 0, len(rows))
	byCode := make(map[string]*MuscleLoad, len(rows))

	for _, row := range rows {
		load, ok := byCode[row.Code]
		if !ok {
			load = &MuscleLoad{Code: row.Code, Name: row.Name, Region: row.Region}
			byCode[row.Code] = load
			order = append(order, row.Code)
		}
		load.Volume += row.Volume * roleWeight(row.Role)
		load.Reps += row.Reps
		load.Sets += row.Sets
	}

	result := make([]MuscleLoad, 0, len(order))
	for _, code := range order {
		result = append(result, *byCode[code])
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Volume > result[j].Volume
	})
	return result
}
