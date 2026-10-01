package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"trainingApp/internal/plan"
	"trainingApp/internal/set"
	"trainingApp/internal/stats"
)

// Стабы хранилищ: вместо генератора моков структура с полями-функциями.
// Тест подставляет только то поведение, которое ему нужно; остальные методы
// возвращают нули и падают, если их вызвали неожиданно.

type setsStoreStub struct {
	t               *testing.T
	createFn        func(ctx context.Context, userID int64, s set.Set) (set.Set, error)
	updateFn        func(ctx context.Context, userID int64, s set.Set) (set.Set, error)
	deleteFn        func(ctx context.Context, userID int64, s set.Set) error
	getByIDFn       func(ctx context.Context, userID, id int64) (set.Set, error)
	listByWorkoutFn func(ctx context.Context, userID, workoutID int64, limit int) ([]set.Set, error)
	lastFn          func(ctx context.Context, userID, exerciseID, excludeWorkoutID int64, planID *int64) (set.LastPerformance, error)
}

func (s setsStoreStub) Create(ctx context.Context, userID int64, v set.Set) (set.Set, error) {
	if s.createFn == nil {
		s.t.Fatal("Create вызван, но не задан в стабе")
	}
	return s.createFn(ctx, userID, v)
}

func (s setsStoreStub) Update(ctx context.Context, userID int64, v set.Set) (set.Set, error) {
	if s.updateFn == nil {
		s.t.Fatal("Update вызван, но не задан в стабе")
	}
	return s.updateFn(ctx, userID, v)
}

func (s setsStoreStub) Delete(ctx context.Context, userID int64, v set.Set) error {
	if s.deleteFn == nil {
		s.t.Fatal("Delete вызван, но не задан в стабе")
	}
	return s.deleteFn(ctx, userID, v)
}

func (s setsStoreStub) GetById(ctx context.Context, userID, id int64) (set.Set, error) {
	if s.getByIDFn == nil {
		s.t.Fatal("GetById вызван, но не задан в стабе")
	}
	return s.getByIDFn(ctx, userID, id)
}

func (s setsStoreStub) ListByWorkout(ctx context.Context, userID, workoutID int64, limit int) ([]set.Set, error) {
	if s.listByWorkoutFn == nil {
		s.t.Fatal("ListByWorkout вызван, но не задан в стабе")
	}
	return s.listByWorkoutFn(ctx, userID, workoutID, limit)
}

func (s setsStoreStub) LastPerformance(ctx context.Context, userID, exerciseID, excludeWorkoutID int64, planID *int64) (set.LastPerformance, error) {
	if s.lastFn == nil {
		s.t.Fatal("LastPerformance вызван, но не задан в стабе")
	}
	return s.lastFn(ctx, userID, exerciseID, excludeWorkoutID, planID)
}

type planStoreStub struct {
	t         *testing.T
	createFn  func(ctx context.Context, p plan.Plan) (plan.Plan, error)
	updateFn  func(ctx context.Context, userID int64, p plan.Plan) (plan.Plan, error)
	listFn    func(ctx context.Context, userID int64, limit int) ([]plan.Plan, error)
	getByIDFn func(ctx context.Context, userID, id int64) (plan.Plan, error)
	deleteFn  func(ctx context.Context, userID, id int64) error
}

func (s planStoreStub) Create(ctx context.Context, p plan.Plan) (plan.Plan, error) {
	if s.createFn == nil {
		s.t.Fatal("Create вызван, но не задан в стабе")
	}
	return s.createFn(ctx, p)
}

func (s planStoreStub) Update(ctx context.Context, userID int64, p plan.Plan) (plan.Plan, error) {
	if s.updateFn == nil {
		s.t.Fatal("Update вызван, но не задан в стабе")
	}
	return s.updateFn(ctx, userID, p)
}

func (s planStoreStub) List(ctx context.Context, userID int64, limit int) ([]plan.Plan, error) {
	if s.listFn == nil {
		s.t.Fatal("List вызван, но не задан в стабе")
	}
	return s.listFn(ctx, userID, limit)
}

func (s planStoreStub) GetByID(ctx context.Context, userID, id int64) (plan.Plan, error) {
	if s.getByIDFn == nil {
		s.t.Fatal("GetByID вызван, но не задан в стабе")
	}
	return s.getByIDFn(ctx, userID, id)
}

func (s planStoreStub) Delete(ctx context.Context, userID, id int64) error {
	if s.deleteFn == nil {
		s.t.Fatal("Delete вызван, но не задан в стабе")
	}
	return s.deleteFn(ctx, userID, id)
}

type statsStoreStub struct {
	t         *testing.T
	muscleFn  func(ctx context.Context, userID int64, p stats.Period) ([]stats.GroupRoleLoad, error)
	recordsFn func(ctx context.Context, userID int64, limit int) ([]stats.Record, error)
	summaryFn func(ctx context.Context, userID int64, p stats.Period) (stats.Summary, error)
}

func (s statsStoreStub) MuscleLoad(ctx context.Context, userID int64, p stats.Period) ([]stats.GroupRoleLoad, error) {
	if s.muscleFn == nil {
		s.t.Fatal("MuscleLoad вызван, но не задан в стабе")
	}
	return s.muscleFn(ctx, userID, p)
}

func (s statsStoreStub) Records(ctx context.Context, userID int64, limit int) ([]stats.Record, error) {
	if s.recordsFn == nil {
		s.t.Fatal("Records вызван, но не задан в стабе")
	}
	return s.recordsFn(ctx, userID, limit)
}

func (s statsStoreStub) Summary(ctx context.Context, userID int64, p stats.Period) (stats.Summary, error) {
	if s.summaryFn == nil {
		s.t.Fatal("Summary вызван, но не задан в стабе")
	}
	return s.summaryFn(ctx, userID, p)
}

// authedRequest собирает запрос с userID в контексте — так же, как это делает
// RequireAuth после разбора токена.
func authedRequest(method, target, body string, userID int64) *http.Request {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, target, nil)
	} else {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	return req.WithContext(context.WithValue(req.Context(), userIDKey, userID))
}
