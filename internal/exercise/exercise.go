package exercise

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrNotFound = errors.New("exercise not found")
var ErrMuscleGroupNotFound = errors.New("muscle group not found")
var ErrAlreadyExists = errors.New("exercise already exists")

const (
	maxExerciseNameLen         = 100
	maxExerciseDescriptionName = 1000
)

type Exercise struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	UserID      *int64    `json:"user_id" db:"user_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

func (e Exercise) Validate() error {
	if strings.TrimSpace(e.Name) == "" {
		return errors.New("name is required")
	}
	if utf8.RuneCountInString(e.Name) > maxExerciseNameLen {
		return errors.New("name must be at most 100 characters")
	}
	if utf8.RuneCountInString(e.Description) > maxExerciseDescriptionName {
		return errors.New("description must be at most 1000 characters")
	}
	return nil
}
