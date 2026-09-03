package set

import "testing"

func TestSetValidate(t *testing.T) {
	tests := []struct {
		name    string
		set     Set
		wantErr bool
	}{
		{name: "обычный подход", set: Set{SetNumber: 1, Reps: 10, Weight: 60}},
		{name: "собственный вес", set: Set{SetNumber: 1, Reps: 12, Weight: 0}},
		{name: "дробный вес", set: Set{SetNumber: 2, Reps: 8, Weight: 11.3}},
		{name: "верхние границы", set: Set{SetNumber: maxSetNumber, Reps: maxRepsNumber, Weight: maxWeight}},

		{name: "нулевой номер подхода", set: Set{SetNumber: 0, Reps: 10, Weight: 60}, wantErr: true},
		{name: "номер подхода выше предела", set: Set{SetNumber: maxSetNumber + 1, Reps: 10, Weight: 60}, wantErr: true},
		{name: "нулевые повторы", set: Set{SetNumber: 1, Reps: 0, Weight: 60}, wantErr: true},
		{name: "повторов больше предела", set: Set{SetNumber: 1, Reps: maxRepsNumber + 1, Weight: 60}, wantErr: true},
		{name: "отрицательный вес", set: Set{SetNumber: 1, Reps: 10, Weight: -1}, wantErr: true},
		{name: "вес выше предела", set: Set{SetNumber: 1, Reps: 10, Weight: maxWeight + 1}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.set.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
