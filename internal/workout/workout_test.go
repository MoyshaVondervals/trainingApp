package workout

import (
	"strings"
	"testing"
	"time"
)

func TestWorkoutValidate(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-2 * time.Hour)
	later := now.Add(time.Hour)

	tests := []struct {
		name    string
		workout Workout
		wantErr bool
	}{
		{name: "начата, не завершена", workout: Workout{StartedAt: now}},
		{name: "завершена позже начала", workout: Workout{StartedAt: earlier, EndedAt: &now}},
		{
			name:    "заметка предельной длины",
			workout: Workout{StartedAt: now, Note: strings.Repeat("я", maxWorkoutNoteLen)},
		},

		{name: "время начала не задано", workout: Workout{}, wantErr: true},
		{
			name:    "начало далеко в будущем",
			workout: Workout{StartedAt: now.Add(time.Hour)},
			wantErr: true,
		},
		{
			name:    "завершение раньше начала",
			workout: Workout{StartedAt: later, EndedAt: &earlier},
			wantErr: true,
		},
		{
			name:    "заметка длиннее предела",
			workout: Workout{StartedAt: now, Note: strings.Repeat("я", maxWorkoutNoteLen+1)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.workout.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestWorkoutValidateAllowsSmallClockSkew(t *testing.T) {
	w := Workout{StartedAt: time.Now().Add(2 * time.Minute)}
	if err := w.Validate(); err != nil {
		t.Fatalf("расхождение часов в пределах допуска должно проходить: %v", err)
	}
}
