package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"trainingApp/internal/plan"
)

func TestPlanHandlerCreate(t *testing.T) {
	validBody := `{"name":"День A","exercises":[{"exercise_id":1,"position":1,"target_sets":4,"target_reps":10}]}`

	tests := []struct {
		name       string
		body       string
		storeErr   error
		wantStatus int
	}{
		{name: "план создан", body: validBody, wantStatus: http.StatusCreated},
		{name: "тело не разбирается", body: `{"name":`, wantStatus: http.StatusBadRequest},
		{
			name:       "без упражнений",
			body:       `{"name":"Пустой","exercises":[]}`,
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "позиция вне диапазона",
			body:       `{"name":"Битый","exercises":[{"exercise_id":1,"position":3,"target_sets":4,"target_reps":10}]}`,
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "упражнение дважды",
			body: `{"name":"Дубли","exercises":[` +
				`{"exercise_id":1,"position":1,"target_sets":4,"target_reps":10},` +
				`{"exercise_id":1,"position":2,"target_sets":4,"target_reps":10}]}`,
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "имя занято",
			body:       validBody,
			storeErr:   plan.ErrAlreadyExists,
			wantStatus: http.StatusConflict,
		},
		{
			name:       "упражнения не существует",
			body:       validBody,
			storeErr:   plan.ErrNotFound,
			wantStatus: http.StatusNotFound,
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
			var gotPlan plan.Plan
			store := planStoreStub{
				t: t,
				createFn: func(_ context.Context, p plan.Plan) (plan.Plan, error) {
					gotPlan = p
					if tt.storeErr != nil {
						return plan.Plan{}, tt.storeErr
					}
					p.ID = 42
					return p, nil
				},
			}

			rec := httptest.NewRecorder()
			NewPlanHandler(store).create(rec, authedRequest(http.MethodPost, "/api/v1/plans", tt.body, 7))

			if rec.Code != tt.wantStatus {
				t.Fatalf("статус = %d, ожидался %d, тело: %s", rec.Code, tt.wantStatus, rec.Body)
			}
			if tt.wantStatus != http.StatusCreated {
				return
			}
			if gotPlan.UserID != 7 {
				t.Errorf("userID в плане = %d, ожидался 7 из контекста", gotPlan.UserID)
			}
			if loc := rec.Header().Get("Location"); loc != "/api/v1/plans/42" {
				t.Errorf("Location = %q, ожидался путь к созданному плану", loc)
			}
		})
	}
}

func TestPlanHandlerGet(t *testing.T) {
	t.Run("план найден", func(t *testing.T) {
		store := planStoreStub{
			t: t,
			getByIDFn: func(_ context.Context, userID, id int64) (plan.Plan, error) {
				if userID != 7 || id != 42 {
					t.Fatalf("в хранилище пришли userID=%d id=%d, ожидались 7 и 42", userID, id)
				}
				return plan.Plan{ID: 42, Name: "День A", Exercises: []plan.Item{
					{ExerciseID: 1, ExerciseName: "Жим", Position: 1, TargetSets: 4, TargetReps: 10},
				}}, nil
			},
		}

		req := authedRequest(http.MethodGet, "/api/v1/plans/42", "", 7)
		req.SetPathValue("id", "42")
		rec := httptest.NewRecorder()
		NewPlanHandler(store).get(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("статус = %d, ожидался 200", rec.Code)
		}
		var got plan.Plan
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("ответ не разбирается: %v", err)
		}
		if len(got.Exercises) != 1 || got.Exercises[0].ExerciseName != "Жим" {
			t.Errorf("состав плана не сериализовался: %+v", got.Exercises)
		}
	})

	t.Run("чужой план", func(t *testing.T) {
		store := planStoreStub{
			t: t,
			getByIDFn: func(context.Context, int64, int64) (plan.Plan, error) {
				return plan.Plan{}, plan.ErrNotFound
			},
		}

		req := authedRequest(http.MethodGet, "/api/v1/plans/42", "", 7)
		req.SetPathValue("id", "42")
		rec := httptest.NewRecorder()
		NewPlanHandler(store).get(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("статус = %d, ожидался 404", rec.Code)
		}
	})

	t.Run("id не число", func(t *testing.T) {
		req := authedRequest(http.MethodGet, "/api/v1/plans/abc", "", 7)
		req.SetPathValue("id", "abc")
		rec := httptest.NewRecorder()
		NewPlanHandler(planStoreStub{t: t}).get(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("статус = %d, ожидался 400 без похода в хранилище", rec.Code)
		}
	})
}

func TestPlanHandlerDelete(t *testing.T) {
	t.Run("план удалён", func(t *testing.T) {
		store := planStoreStub{
			t:        t,
			deleteFn: func(context.Context, int64, int64) error { return nil },
		}

		req := authedRequest(http.MethodDelete, "/api/v1/plans/42", "", 7)
		req.SetPathValue("id", "42")
		rec := httptest.NewRecorder()
		NewPlanHandler(store).delete(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("статус = %d, ожидался 204", rec.Code)
		}
		if rec.Body.Len() != 0 {
			t.Errorf("тело ответа не пустое: %q", rec.Body)
		}
	})

	t.Run("повторное удаление", func(t *testing.T) {
		store := planStoreStub{
			t:        t,
			deleteFn: func(context.Context, int64, int64) error { return plan.ErrNotFound },
		}

		req := authedRequest(http.MethodDelete, "/api/v1/plans/42", "", 7)
		req.SetPathValue("id", "42")
		rec := httptest.NewRecorder()
		NewPlanHandler(store).delete(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("статус = %d, ожидался 404", rec.Code)
		}
	})
}
