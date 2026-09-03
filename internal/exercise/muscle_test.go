package exercise

import (
	"strings"
	"testing"
)

func TestValidateMuscles(t *testing.T) {
	tests := []struct {
		name    string
		muscles []Muscle
		wantErr bool
	}{
		{
			name:    "один primary",
			muscles: []Muscle{{MuscleGroupID: 1, Role: RolePrimary}},
		},
		{
			name: "primary и несколько secondary",
			muscles: []Muscle{
				{MuscleGroupID: 1, Role: RolePrimary},
				{MuscleGroupID: 2, Role: RoleSecondary},
				{MuscleGroupID: 3, Role: RoleSecondary},
			},
		},
		{
			name:    "пустой набор",
			muscles: []Muscle{},
			wantErr: true,
		},
		{
			name: "два primary",
			muscles: []Muscle{
				{MuscleGroupID: 1, Role: RolePrimary},
				{MuscleGroupID: 2, Role: RolePrimary},
			},
			wantErr: true,
		},
		{
			name:    "ни одного primary",
			muscles: []Muscle{{MuscleGroupID: 1, Role: RoleSecondary}},
			wantErr: true,
		},
		{
			name: "дубликат группы",
			muscles: []Muscle{
				{MuscleGroupID: 1, Role: RolePrimary},
				{MuscleGroupID: 1, Role: RoleSecondary},
			},
			wantErr: true,
		},
		{
			name:    "неизвестная роль",
			muscles: []Muscle{{MuscleGroupID: 1, Role: "main"}},
			wantErr: true,
		},
		{
			name:    "нулевой идентификатор группы",
			muscles: []Muscle{{MuscleGroupID: 0, Role: RolePrimary}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMuscles(tt.muscles)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateMuscles() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateMusclesLimit(t *testing.T) {
	muscles := make([]Muscle, 0, maxMusclesPerExercise+1)
	muscles = append(muscles, Muscle{MuscleGroupID: 1, Role: RolePrimary})
	for i := 2; i <= maxMusclesPerExercise; i++ {
		muscles = append(muscles, Muscle{MuscleGroupID: int64(i), Role: RoleSecondary})
	}
	if err := ValidateMuscles(muscles); err != nil {
		t.Fatalf("%d групп должно проходить: %v", maxMusclesPerExercise, err)
	}

	muscles = append(muscles, Muscle{MuscleGroupID: maxMusclesPerExercise + 1, Role: RoleSecondary})
	if err := ValidateMuscles(muscles); err == nil {
		t.Fatalf("%d групп должно отклоняться", maxMusclesPerExercise+1)
	}
}

func TestValidateGroupCode(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{name: "обычный код", code: "biceps"},
		{name: "пустая строка", code: "", wantErr: true},
		{name: "только пробелы", code: "   ", wantErr: true},
		{name: "предельная длина", code: strings.Repeat("a", maxMuscleCodeLen)},
		{name: "длиннее предела", code: strings.Repeat("a", maxMuscleCodeLen+1), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGroupCode(tt.code)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateGroupCode(%q) error = %v, wantErr = %v", tt.code, err, tt.wantErr)
			}
		})
	}
}
