package api

import (
	"net/http"

	"trainingApp/internal/exercise"
)

type Handler struct {
	exercises *exercise.Store
}

func New(exercises *exercise.Store) *Handler {
	return &Handler{exercises: exercises}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("POST /api/v1/exercises", h.createExercise)
	mux.HandleFunc("GET /api/v1/exercises", h.listExercises)
	mux.HandleFunc("GET /api/v1/exercises/{id}", h.getExercise)

	return mux
}
