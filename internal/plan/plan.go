package plan

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrNotFound = errors.New("plan not found")
var ErrAlreadyExists = errors.New("plan already exists")

const (
	maxPlanNameLen = 100
	maxPlanNoteLen = 1000
	maxPlanItems   = 30
	maxTargetSets  = 30
	maxTargetReps  = 5000
)

type Item struct {
	ExerciseID   int64  `json:"exercise_id" db:"exercise_id"`
	ExerciseName string `json:"exercise_name,omitempty" db:"exercise_name"`
	Position     int    `json:"position" db:"position"`
	TargetSets   int    `json:"target_sets" db:"target_sets"`
	TargetReps   int    `json:"target_reps" db:"target_reps"`
}

type Plan struct {
	ID        int64     `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	Name      string    `json:"name" db:"name"`
	Note      string    `json:"note" db:"note"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	Exercises []Item    `json:"exercises"`
}

func (p Plan) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("name is required")
	}
	if n := utf8.RuneCountInString(p.Name); n > maxPlanNameLen {
		return fmt.Errorf("name must be at most %d characters, got %d", maxPlanNameLen, n)
	}
	if n := utf8.RuneCountInString(p.Note); n > maxPlanNoteLen {
		return fmt.Errorf("note must be at most %d characters, got %d", maxPlanNoteLen, n)
	}
	return ValidateItems(p.Exercises)
}

func ValidateItems(items []Item) error {
	if len(items) == 0 {
		return errors.New("at least one exercise is required")
	}
	if len(items) > maxPlanItems {
		return fmt.Errorf("at most %d exercises allowed, got %d", maxPlanItems, len(items))
	}

	seenExercise := make(map[int64]struct{}, len(items))
	seenPosition := make(map[int]struct{}, len(items))

	for i, item := range items {
		if item.ExerciseID < 1 {
			return fmt.Errorf("exercise #%d: exercise_id is required", i+1)
		}
		if _, dup := seenExercise[item.ExerciseID]; dup {
			return fmt.Errorf("exercise %d is listed twice", item.ExerciseID)
		}
		seenExercise[item.ExerciseID] = struct{}{}

		if item.Position < 1 || item.Position > len(items) {
			return fmt.Errorf("exercise #%d: position must be between 1 and %d", i+1, len(items))
		}
		if _, dup := seenPosition[item.Position]; dup {
			return fmt.Errorf("position %d is used twice", item.Position)
		}
		seenPosition[item.Position] = struct{}{}

		if item.TargetSets < 1 || item.TargetSets > maxTargetSets {
			return fmt.Errorf("exercise #%d: target_sets must be between 1 and %d", i+1, maxTargetSets)
		}
		if item.TargetReps < 1 || item.TargetReps > maxTargetReps {
			return fmt.Errorf("exercise #%d: target_reps must be between 1 and %d", i+1, maxTargetReps)
		}
	}
	return nil
}
