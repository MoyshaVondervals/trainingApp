package weight

import (
	"errors"
	"time"
	"unicode/utf8"
)

var ErrNotFound = errors.New("body weight not found")

const (
	minWeightKg = 20
	maxWeightKg = 500
	maxNoteLen  = 200
)

type BodyWeight struct {
	ID         int64     `json:"id" db:"id"`
	UserID     int64     `json:"user_id" db:"user_id"`
	WeightKg   float32   `json:"weight_kg" db:"weight_kg"`
	MeasuredOn time.Time `json:"measured_on" db:"measured_on"`
	Note       string    `json:"note" db:"note"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

func (b BodyWeight) Validate() error {
	if b.WeightKg < minWeightKg || b.WeightKg > maxWeightKg {
		return errors.New("weight_kg has to be between 20 and 500")
	}
	if b.MeasuredOn.IsZero() {
		return errors.New("measured_on is required")
	}
	if b.MeasuredOn.After(time.Now().AddDate(0, 0, 1)) {
		return errors.New("measured_on is in the future")
	}
	if utf8.RuneCountInString(b.Note) > maxNoteLen {
		return errors.New("note is too long Max len 200")
	}
	return nil
}
