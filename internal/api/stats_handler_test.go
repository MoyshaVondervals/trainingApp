package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
	"trainingApp/internal/stats"
)

func TestStatsHandlerDashboard(t *testing.T) {
	store := statsStoreStub{
		t: t,
		summaryFn: func(context.Context, int64, stats.Period) (stats.Summary, error) {
			return stats.Summary{Workouts: 2, Sets: 10, Reps: 100, Volume: 4000}, nil
		},
		muscleFn: func(context.Context, int64, stats.Period) ([]stats.GroupRoleLoad, error) {
			return []stats.GroupRoleLoad{
				{Code: "lats", Name: "Широчайшие", Region: "Спина", Role: stats.RolePrimary, Volume: 1000, Reps: 40, Sets: 4},
				{Code: "lats", Name: "Широчайшие", Region: "Спина", Role: "secondary", Volume: 400, Reps: 20, Sets: 2},
			}, nil
		},
		recordsFn: func(context.Context, int64, int) ([]stats.Record, error) {
			return []stats.Record{{ExerciseID: 1, ExerciseName: "Жим", Reps: 10}}, nil
		},
	}

	rec := httptest.NewRecorder()
	NewStatsHandler(store, nil).dashboard(rec, authedRequest(http.MethodGet, "/api/v1/stats", "", 7))

	if rec.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидался 200, тело: %s", rec.Code, rec.Body)
	}

	var got stats.Dashboard
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("ответ не разбирается: %v", err)
	}
	if got.Summary.Volume != 4000 {
		t.Errorf("volume = %v, ожидалось 4000", got.Summary.Volume)
	}
	if len(got.Muscles) != 1 {
		t.Fatalf("групп в ответе = %d, ожидалась одна: роли сворачиваются в одну строку", len(got.Muscles))
	}
	// 1000 с коэффициентом 1,0 плюс 400 с коэффициентом 0,5 для secondary.
	if got.Muscles[0].Volume != 1200 {
		t.Errorf("volume группы = %v, ожидалось 1200", got.Muscles[0].Volume)
	}
	if got.Period.To.Sub(got.Period.From) <= 0 {
		t.Error("период пустой: по умолчанию ожидается непустой интервал")
	}
}

func TestStatsHandlerPeriodValidation(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		wantStatus int
	}{
		{name: "границы не заданы", target: "/api/v1/stats", wantStatus: http.StatusOK},
		{
			name:       "корректный период",
			target:     "/api/v1/stats?from=2026-08-01T00:00:00Z&to=2026-09-01T00:00:00Z",
			wantStatus: http.StatusOK,
		},
		{name: "from не дата", target: "/api/v1/stats?from=вчера", wantStatus: http.StatusBadRequest},
		{
			name:       "начало позже конца",
			target:     "/api/v1/stats?from=2026-09-01T00:00:00Z&to=2026-08-01T00:00:00Z",
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := statsStoreStub{
				t:         t,
				summaryFn: func(context.Context, int64, stats.Period) (stats.Summary, error) { return stats.Summary{}, nil },
				muscleFn: func(context.Context, int64, stats.Period) ([]stats.GroupRoleLoad, error) {
					return nil, nil
				},
				recordsFn: func(context.Context, int64, int) ([]stats.Record, error) { return nil, nil },
			}

			rec := httptest.NewRecorder()
			NewStatsHandler(store, nil).dashboard(rec, authedRequest(http.MethodGet, tt.target, "", 7))

			if rec.Code != tt.wantStatus {
				t.Fatalf("статус = %d, ожидался %d, тело: %s", rec.Code, tt.wantStatus, rec.Body)
			}
		})
	}
}

// Три выборки идут параллельно через errgroup. Тест проверяет две вещи сразу:
// что они действительно запускаются одновременно, и что ошибка любой из них
// превращается в 500 вместо частичного ответа.
func TestStatsHandlerRunsQueriesConcurrently(t *testing.T) {
	var running, maxRunning atomic.Int32

	track := func() func() {
		cur := running.Add(1)
		for {
			old := maxRunning.Load()
			if cur <= old || maxRunning.CompareAndSwap(old, cur) {
				break
			}
		}
		time.Sleep(30 * time.Millisecond)
		return func() { running.Add(-1) }
	}

	store := statsStoreStub{
		t: t,
		summaryFn: func(context.Context, int64, stats.Period) (stats.Summary, error) {
			defer track()()
			return stats.Summary{}, nil
		},
		muscleFn: func(context.Context, int64, stats.Period) ([]stats.GroupRoleLoad, error) {
			defer track()()
			return nil, nil
		},
		recordsFn: func(context.Context, int64, int) ([]stats.Record, error) {
			defer track()()
			return nil, nil
		},
	}

	start := time.Now()
	rec := httptest.NewRecorder()
	NewStatsHandler(store, nil).dashboard(rec, authedRequest(http.MethodGet, "/api/v1/stats", "", 7))
	elapsed := time.Since(start)

	if rec.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидался 200", rec.Code)
	}
	if got := maxRunning.Load(); got != 3 {
		t.Errorf("одновременно работало %d выборок, ожидалось 3", got)
	}
	if elapsed > 80*time.Millisecond {
		t.Errorf("обработка заняла %v: похоже, выборки идут последовательно", elapsed)
	}
}

func TestStatsHandlerFailsWholeDashboardOnQueryError(t *testing.T) {
	store := statsStoreStub{
		t: t,
		summaryFn: func(context.Context, int64, stats.Period) (stats.Summary, error) {
			return stats.Summary{Workouts: 1}, nil
		},
		muscleFn: func(context.Context, int64, stats.Period) ([]stats.GroupRoleLoad, error) {
			return nil, errors.New("deadlock detected")
		},
		recordsFn: func(context.Context, int64, int) ([]stats.Record, error) { return nil, nil },
	}

	rec := httptest.NewRecorder()
	NewStatsHandler(store, nil).dashboard(rec, authedRequest(http.MethodGet, "/api/v1/stats", "", 7))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("статус = %d, ожидался 500: частичный дашборд отдавать нельзя", rec.Code)
	}
	if body := rec.Body.String(); !json.Valid(rec.Body.Bytes()) || body == "" {
		t.Errorf("ожидался JSON с ошибкой, получено: %q", body)
	}
}

type dashboardCacheStub struct {
	get func(context.Context, int64, string, string) (stats.Dashboard, bool, error)
	set func(context.Context, int64, string, string, stats.Dashboard) error
}

func (s dashboardCacheStub) Get(ctx context.Context, userID int64, from, to string) (stats.Dashboard, bool, error) {
	return s.get(ctx, userID, from, to)
}

func (s dashboardCacheStub) Set(ctx context.Context, userID int64, from, to string, value stats.Dashboard) error {
	return s.set(ctx, userID, from, to, value)
}

func TestStatsHandlerCacheHitSkipsDatabase(t *testing.T) {
	const target = "/api/v1/stats?from=2026-08-01T00:00:00Z&to=2026-09-01T00:00:00Z"
	cache := dashboardCacheStub{
		get: func(_ context.Context, userID int64, from, to string) (stats.Dashboard, bool, error) {
			if userID != 7 || from != "2026-08-01T00:00:00Z" || to != "2026-09-01T00:00:00Z" {
				t.Fatalf("неверный ключ кэша: user=%d from=%q to=%q", userID, from, to)
			}
			return stats.Dashboard{Summary: stats.Summary{Workouts: 3}}, true, nil
		},
		set: func(context.Context, int64, string, string, stats.Dashboard) error {
			t.Fatal("при попадании в кэш запись не нужна")
			return nil
		},
	}
	rec := httptest.NewRecorder()
	NewStatsHandler(statsStoreStub{t: t}, cache).dashboard(rec, authedRequest(http.MethodGet, target, "", 7))
	if rec.Code != http.StatusOK {
		t.Fatalf("статус = %d, тело: %s", rec.Code, rec.Body)
	}
	var got stats.Dashboard
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Summary.Workouts != 3 {
		t.Fatalf("ответ из кэша: %+v, ошибка: %v", got, err)
	}
}

func TestStatsHandlerCacheMissStoresDashboard(t *testing.T) {
	const target = "/api/v1/stats?from=2026-08-01T00:00:00Z&to=2026-09-01T00:00:00Z"
	var saved stats.Dashboard
	cache := dashboardCacheStub{
		get: func(context.Context, int64, string, string) (stats.Dashboard, bool, error) {
			return stats.Dashboard{}, false, nil
		},
		set: func(_ context.Context, userID int64, from, to string, value stats.Dashboard) error {
			if userID != 7 || from != "2026-08-01T00:00:00Z" || to != "2026-09-01T00:00:00Z" {
				t.Fatalf("неверный ключ записи: user=%d from=%q to=%q", userID, from, to)
			}
			saved = value
			return nil
		},
	}
	store := statsStoreStub{
		t: t,
		summaryFn: func(context.Context, int64, stats.Period) (stats.Summary, error) {
			return stats.Summary{Workouts: 2}, nil
		},
		muscleFn:  func(context.Context, int64, stats.Period) ([]stats.GroupRoleLoad, error) { return nil, nil },
		recordsFn: func(context.Context, int64, int) ([]stats.Record, error) { return nil, nil },
	}
	rec := httptest.NewRecorder()
	NewStatsHandler(store, cache).dashboard(rec, authedRequest(http.MethodGet, target, "", 7))
	if rec.Code != http.StatusOK || saved.Summary.Workouts != 2 {
		t.Fatalf("статус = %d, сохранено: %+v, тело: %s", rec.Code, saved, rec.Body)
	}
}
