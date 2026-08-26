package api

import (
	"net/http"
)

func Router(ex *ExerciseHandler, users *UserHandler, wo *WorkoutsHandler, s *SetHandler, em *ExerciseMuscleHandler, st *StatsHandler) http.Handler {
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
	protected.HandleFunc("GET /api/v1/exercises/{id}/muscles", em.list)
	protected.HandleFunc("PUT /api/v1/exercises/{id}/muscles", em.replace)

	protected.HandleFunc("POST /api/v1/workouts", wo.startTraining)
	protected.HandleFunc("GET /api/v1/workouts/{id}", wo.getTraining)
	protected.HandleFunc("GET /api/v1/workouts", wo.listWorkout)
	protected.HandleFunc("PATCH /api/v1/workouts/{id}", wo.updateNote)
	protected.HandleFunc("POST /api/v1/workouts/finish/{id}", wo.finishTraining)
	protected.HandleFunc("DELETE /api/v1/workouts/{id}", wo.deleteTraining)

	protected.HandleFunc("POST /api/v1/sets", s.create)
	protected.HandleFunc("PATCH /api/v1/sets/{id}", s.update)
	protected.HandleFunc("DELETE /api/v1/sets/{id}", s.delete)
	protected.HandleFunc("GET /api/v1/sets/{id}", s.getById)
	protected.HandleFunc("GET /api/v1/sets/byWorkout/{id}", s.listByWorkout)
	protected.HandleFunc("GET /api/v1/stats", st.dashboard)

	mux.Handle("/api/v1/", users.RequireAuth(protected))

	return mux
}
