package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"trainingApp/internal/set"
)

func TestSetHandlerCreate(t *testing.T) {
	validBody := `{"exercise_id":1,"workout_id":2,"set_number":1,"reps":10,"weight":42.5}`

	tests := []struct {
		name       string
		body       string
		storeErr   error
		wantStatus int
		wantError  string
	}{
		{
			name:       "подход создан",
			body:       validBody,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "тело не разбирается",
			body:       `{"exercise_id":`,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid request body",
		},
		{
			name:       "неизвестное поле отклоняется",
			body:       `{"exercise_id":1,"workout_id":2,"set_number":1,"reps":10,"weight":10,"extra":true}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "повторов больше предела",
			body:       `{"exercise_id":1,"workout_id":2,"set_number":1,"reps":99999,"weight":10}`,
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "чужая тренировка",
			body:       validBody,
			storeErr:   set.ErrNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "номер подхода занят",
			body:       validBody,
			storeErr:   set.ErrAlreadyExists,
			wantStatus: http.StatusConflict,
		},
		{
			name:       "ошибка хранилища",
			body:       validBody,
			storeErr:   errors.New("connection refused"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUserID int64
			store := setsStoreStub{
				t: t,
				createFn: func(_ context.Context, userID int64, s set.Set) (set.Set, error) {
					gotUserID = userID
					if tt.storeErr != nil {
						return set.Set{}, tt.storeErr
					}
					s.ID = 100
					return s, nil
				},
			}
			h := NewSetHandler(store)

			rec := httptest.NewRecorder()
			h.create(rec, authedRequest(http.MethodPost, "/api/v1/sets", tt.body, 7))

			if rec.Code != tt.wantStatus {
				t.Fatalf("статус = %d, ожидался %d, тело: %s", rec.Code, tt.wantStatus, rec.Body)
			}

			if tt.wantStatus == http.StatusCreated {
				if gotUserID != 7 {
					t.Errorf("в хранилище передан userID %d, ожидался 7 из контекста", gotUserID)
				}
				if loc := rec.Header().Get("Location"); loc != "/api/v1/sets/100" {
					t.Errorf("Location = %q, ожидался путь к созданному подходу", loc)
				}
				var created set.Set
				if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
					t.Fatalf("ответ не разбирается: %v", err)
				}
				if created.ID != 100 {
					t.Errorf("id в ответе = %d, ожидался 100", created.ID)
				}
			}

			if tt.wantError != "" {
				var body struct {
					Error string `json:"error"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("ответ об ошибке не разбирается: %v", err)
				}
				if body.Error != tt.wantError {
					t.Errorf("error = %q, ожидалось %q", body.Error, tt.wantError)
				}
			}
		})
	}
}

func TestSetHandlerCreateWithoutUserInContext(t *testing.T) {
	h := NewSetHandler(setsStoreStub{t: t})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sets",
		strings.NewReader(`{"exercise_id":1,"workout_id":2,"set_number":1,"reps":10,"weight":10}`))
	h.create(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("статус = %d, ожидался 401: без userID в контексте хендлер не должен ходить в хранилище", rec.Code)
	}
}

func TestSetHandlerLastByExercise(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		wantPlanID *int64
		wantStatus int
	}{
		{name: "без параметров", target: "/api/v1/sets/last/5", wantStatus: http.StatusOK},
		{
			name:       "с планом",
			target:     "/api/v1/sets/last/5?exclude_workout=3&plan_id=9",
			wantPlanID: ptr(int64(9)),
			wantStatus: http.StatusOK,
		},
		{name: "plan_id не число", target: "/api/v1/sets/last/5?plan_id=abc", wantStatus: http.StatusBadRequest},
		{name: "exclude_workout не число", target: "/api/v1/sets/last/5?exclude_workout=x", wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPlanID *int64
			store := setsStoreStub{
				t: t,
				lastFn: func(_ context.Context, _, _, _ int64, planID *int64) (set.LastPerformance, error) {
					gotPlanID = planID
					return set.LastPerformance{WorkoutID: 3}, nil
				},
			}
			h := NewSetHandler(store)

			req := authedRequest(http.MethodGet, tt.target, "", 7)
			req.SetPathValue("id", "5")
			rec := httptest.NewRecorder()
			h.lastByExercise(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("статус = %d, ожидался %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			switch {
			case tt.wantPlanID == nil && gotPlanID != nil:
				t.Errorf("plan_id = %d, ожидался nil", *gotPlanID)
			case tt.wantPlanID != nil && gotPlanID == nil:
				t.Errorf("plan_id = nil, ожидался %d", *tt.wantPlanID)
			case tt.wantPlanID != nil && *gotPlanID != *tt.wantPlanID:
				t.Errorf("plan_id = %d, ожидался %d", *gotPlanID, *tt.wantPlanID)
			}
		})
	}
}

func TestSetHandlerLastByExerciseNotFound(t *testing.T) {
	store := setsStoreStub{
		t: t,
		lastFn: func(context.Context, int64, int64, int64, *int64) (set.LastPerformance, error) {
			return set.LastPerformance{}, set.ErrNotFound
		},
	}
	h := NewSetHandler(store)

	req := authedRequest(http.MethodGet, "/api/v1/sets/last/5", "", 7)
	req.SetPathValue("id", "5")
	rec := httptest.NewRecorder()
	h.lastByExercise(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("статус = %d, ожидался 404 для упражнения без истории", rec.Code)
	}
}

func ptr[T any](v T) *T { return &v }
