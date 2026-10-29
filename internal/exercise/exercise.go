package exercise

import (
	"errors"
	"fmt"
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
	maxExerciseLinkLen         = 500
)

type Exercise struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	Link        string    `json:"link,omitempty" db:"link"`
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
	if err := validateLink(e.Link); err != nil {
		return err
	}
	if utf8.RuneCountInString(e.Description) > maxExerciseDescriptionName {
		return errors.New("description must be at most 1000 characters")
	}
	return nil
}

func validateLink(link string) error {
	if link == "" {
		return nil
	}
	if n := utf8.RuneCountInString(link); n > maxExerciseLinkLen {
		return fmt.Errorf("link must be at most %d characters, got %d", maxExerciseLinkLen, n)
	}
	if !strings.HasPrefix(link, "https://") && !strings.HasPrefix(link, "http://") {
		return errors.New("link must start with http:// or https://")
	}
	return nil
}
