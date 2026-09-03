package weight

import (
	"strings"
	"testing"
	"time"
)

func TestBodyWeightValidate(t *testing.T) {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	tests := []struct {
		name    string
		weight  BodyWeight
		wantErr bool
	}{
		{name: "обычный замер", weight: BodyWeight{WeightKg: 76.6, MeasuredOn: today}},
		{name: "нижняя граница", weight: BodyWeight{WeightKg: minWeightKg, MeasuredOn: today}},
		{name: "верхняя граница", weight: BodyWeight{WeightKg: maxWeightKg, MeasuredOn: today}},
		{name: "прошлая дата", weight: BodyWeight{WeightKg: 80, MeasuredOn: today.AddDate(0, -3, 0)}},
		{
			name:   "заметка предельной длины",
			weight: BodyWeight{WeightKg: 80, MeasuredOn: today, Note: strings.Repeat("я", maxNoteLen)},
		},

		{name: "вес ниже предела", weight: BodyWeight{WeightKg: minWeightKg - 1, MeasuredOn: today}, wantErr: true},
		{name: "вес выше предела", weight: BodyWeight{WeightKg: maxWeightKg + 1, MeasuredOn: today}, wantErr: true},
		{name: "нулевой вес", weight: BodyWeight{WeightKg: 0, MeasuredOn: today}, wantErr: true},
		{name: "дата не задана", weight: BodyWeight{WeightKg: 80}, wantErr: true},
		{
			name:    "дата в будущем",
			weight:  BodyWeight{WeightKg: 80, MeasuredOn: today.AddDate(0, 0, 5)},
			wantErr: true,
		},
		{
			name:    "заметка длиннее предела",
			weight:  BodyWeight{WeightKg: 80, MeasuredOn: today, Note: strings.Repeat("я", maxNoteLen+1)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.weight.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
