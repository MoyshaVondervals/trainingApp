package set

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("set not found")
var ErrAlreadyExists = errors.New("set number already used")

const (
	maxSetNumber  = 30
	maxRepsNumber = 5000 // Я отжимался полторы тысячи раз и приседал по 3 тысячи раз хахахаххаха
	maxWeight     = 350
)

type Set struct {
	ID         int64     `json:"id" db:"id"`
	ExerciseID int64     `json:"exercise_id" db:"exercise_id"`
	WorkoutID  int64     `json:"workout_id" db:"workout_id"`
	SetNumber  int64     `json:"set_number" db:"set_number"`
	Reps       int       `json:"reps" db:"reps"`
	Weight     float32   `json:"weight" db:"weight_kg"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

func (s *Set) Validate() error {
	if s.SetNumber < 1 || s.SetNumber > maxSetNumber {
		return errors.New("invalid set number it has to be between 1 and 30")
	}
	if s.Reps < 1 || s.Reps > maxRepsNumber {
		return errors.New("invalid set reps it has to be between 1 and 5000")
	}
	if s.Weight < 0 || s.Weight > maxWeight {
		return errors.New("invalid set weight it has to be between 0 and 350")
	}
	return nil
}

type LastPerformance struct {
	WorkoutID   int64     `json:"workout_id" db:"workout_id"`
	PerformedAt time.Time `json:"performed_at" db:"performed_at"`
	Sets        []Set     `json:"sets"`
}
