package api

import (
	"context"
	"net/http"
	"trainingApp/internal/exercise"
)

type Handler struct {
	exercises ExerciseStore
}

type ExerciseStore interface {
	Create(ctx context.Context, e exercise.Exercise) (exercise.Exercise, error)
	GetByID(ctx context.Context, id int64) (exercise.Exercise, error)
	List(ctx context.Context, limit int) ([]exercise.Exercise, error)
}

func New(store ExerciseStore) *Handler {
	return &Handler{exercises: store}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("POST /api/v1/exercises", h.createExercise)
	mux.HandleFunc("GET /api/v1/exercises", h.listExercises)
	mux.HandleFunc("GET /api/v1/exercises/{id}", h.getExercise)

	return mux
}
