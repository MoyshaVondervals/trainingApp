package exercise

import (
	"errors"
	"fmt"
	"strings"
)

const (
	RolePrimary   = "primary"
	RoleSecondary = "secondary"
)

const (
	maxMusclesPerExercise = 20
	maxMuscleCodeLen      = 50
)

type Muscle struct {
	MuscleGroupID int64  `json:"muscle_group_id" db:"muscle_group_id"`
	Role          string `json:"role" db:"role"`
}

func (m Muscle) Validate() error {
	if m.MuscleGroupID < 1 {
		return errors.New("muscle_group_id is required")
	}
	if m.Role != RolePrimary && m.Role != RoleSecondary {
		return fmt.Errorf("role must be %q or %q, got %q", RolePrimary, RoleSecondary, m.Role)
	}
	return nil
}

func ValidateMuscles(muscles []Muscle) error {
	if len(muscles) == 0 {
		return errors.New("at least one muscle group is required")
	}
	if len(muscles) > maxMusclesPerExercise {
		return fmt.Errorf("at most %d muscle groups allowed, got %d", maxMusclesPerExercise, len(muscles))
	}

	seen := make(map[int64]struct{}, len(muscles))
	primaries := 0

	for i, m := range muscles {
		if err := m.Validate(); err != nil {
			return fmt.Errorf("muscle #%d: %w", i+1, err)
		}
		if _, dup := seen[m.MuscleGroupID]; dup {
			return fmt.Errorf("muscle group %d is listed twice", m.MuscleGroupID)
		}
		seen[m.MuscleGroupID] = struct{}{}
		if m.Role == RolePrimary {
			primaries++
		}
	}

	if primaries == 0 {
		return errors.New("exactly one muscle group must have role \"primary\", got none")
	}
	if primaries > 1 {
		return fmt.Errorf("exactly one muscle group must have role %q, got %d", RolePrimary, primaries)
	}
	return nil
}

type WithRole struct {
	Exercise
	Role string `json:"role" db:"role"`
}

func ValidateGroupCode(code string) error {
	if strings.TrimSpace(code) == "" {
		return errors.New("muscle group code is required")
	}
	if len(code) > maxMuscleCodeLen {
		return fmt.Errorf("muscle group code must be at most %d characters", maxMuscleCodeLen)
	}
	return nil
}

type Group struct {
	ID         int64  `json:"id" db:"id"`
	Code       string `json:"code" db:"code"`
	Name       string `json:"name" db:"name"`
	RegionCode string `json:"region_code" db:"region_code"`
	RegionName string `json:"region_name" db:"region_name"`
}
