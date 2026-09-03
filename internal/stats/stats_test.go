package stats

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestAggregateByGroupRoleWeights(t *testing.T) {
	rows := []GroupRoleLoad{
		{Code: "lats", Name: "Широчайшие", Region: "Спина", Role: RolePrimary, Volume: 1000, Reps: 40, Sets: 4},
		{Code: "lats", Name: "Широчайшие", Region: "Спина", Role: "secondary", Volume: 400, Reps: 20, Sets: 2},
	}

	got := AggregateByGroup(rows)
	if len(got) != 1 {
		t.Fatalf("ожидалась одна группа, получено %d", len(got))
	}

	const want = 1000*1.0 + 400*0.5
	if math.Abs(got[0].Volume-want) > 1e-9 {
		t.Errorf("Volume = %v, ожидалось %v", got[0].Volume, want)
	}
	if got[0].Reps != 60 {
		t.Errorf("Reps = %d, ожидалось 60", got[0].Reps)
	}
	if got[0].Sets != 6 {
		t.Errorf("Sets = %d, ожидалось 6", got[0].Sets)
	}
}

func TestAggregateByGroupSortsByVolumeDesc(t *testing.T) {
	rows := []GroupRoleLoad{
		{Code: "biceps", Role: "secondary", Volume: 200},
		{Code: "chest", Role: RolePrimary, Volume: 900},
		{Code: "triceps", Role: RolePrimary, Volume: 500},
	}

	got := AggregateByGroup(rows)
	wantOrder := []string{"chest", "triceps", "biceps"}
	for i, code := range wantOrder {
		if got[i].Code != code {
			t.Fatalf("позиция %d: код %q, ожидался %q", i, got[i].Code, code)
		}
	}
}

func TestAggregateByGroupEmpty(t *testing.T) {
	if got := AggregateByGroup(nil); len(got) != 0 {
		t.Fatalf("для пустого входа ожидался пустой срез, получено %d", len(got))
	}
}

func TestNewPeriod(t *testing.T) {
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	t.Run("обе границы заданы", func(t *testing.T) {
		p, err := NewPeriod(from, to)
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if !p.From.Equal(from) || !p.To.Equal(to) {
			t.Fatalf("границы изменены: %v — %v", p.From, p.To)
		}
	})

	t.Run("границы не заданы", func(t *testing.T) {
		p, err := NewPeriod(time.Time{}, time.Time{})
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if got := p.To.Sub(p.From); got != defaultPeriod {
			t.Fatalf("длина периода по умолчанию = %v, ожидалось %v", got, defaultPeriod)
		}
	})

	t.Run("начало позже конца", func(t *testing.T) {
		if _, err := NewPeriod(to, from); !errors.Is(err, ErrBadPeriod) {
			t.Fatalf("error = %v, ожидалась ErrBadPeriod", err)
		}
	})

	t.Run("границы совпадают", func(t *testing.T) {
		if _, err := NewPeriod(from, from); !errors.Is(err, ErrBadPeriod) {
			t.Fatalf("error = %v, ожидалась ErrBadPeriod", err)
		}
	})
}
