package api

import (
	"net/http"
)

func Router(ex *ExerciseHandler, users *UserHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("POST /api/v1/auth/register", users.registerUser)
	mux.HandleFunc("POST /api/v1/auth/login", users.loginUser)

	protected := http.NewServeMux()
	protected.HandleFunc("GET /api/v1/exercises", ex.listExercises)
	protected.HandleFunc("POST /api/v1/exercises", ex.createExercise)
	protected.HandleFunc("PATCH /api/v1/exercises/{id}", ex.updateExercise)
	protected.HandleFunc("DELETE /api/v1/exercises/{id}", ex.deleteExercise)
	protected.HandleFunc("GET /api/v1/exercises/{id}", ex.getExercise)

	mux.Handle("/api/v1/", users.RequireAuth(protected))

	return mux
}
