package workout

import (
	"errors"
	"time"
	"unicode/utf8"
)

var ErrNotFound = errors.New("workout not found")

const (
	maxWorkoutNoteLen = 1000
)

type Workout struct {
	ID        int64      `json:"id" db:"id"`
	UserID    int64      `json:"user_id" db:"user_id"`
	StartedAt time.Time  `json:"started_at" db:"started_at"`
	EndedAt   *time.Time `json:"ended_at" db:"ended_at"`
	Note      string     `json:"note" db:"note"`
}

func (e Workout) Validate() error {
	if err := e.ValidateNote(); err != nil {
		return err
	}
	if e.StartedAt.IsZero() {
		return errors.New("started_at is required")
	}
	if e.StartedAt.After(time.Now().Add(5 * time.Minute)) {
		return errors.New("started_at is in the future")
	}
	if e.EndedAt != nil && e.EndedAt.Before(e.StartedAt) {
		return errors.New("ended_at must be earlier than started_at")
	}
	return nil
}

func (e Workout) ValidateNote() error {
	if utf8.RuneCountInString(e.Note) > maxWorkoutNoteLen {
		return errors.New("note is too long Max len 1000")
	}
	return nil
}
