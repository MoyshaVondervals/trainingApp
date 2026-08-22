package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"trainingApp/internal/exercise"
)

type ExerciseMuscleHandler struct {
	exerciseMuscle ExerciseMuscleStore
}

type ExerciseMuscleStore interface {
	ListByExercise(ctx context.Context, userID, exerciseID int64) ([]exercise.Muscle, error)
	ReplaceForExercise(ctx context.Context, userID, exerciseID int64, muscles []exercise.Muscle) error
}

func NewExerciseMuscleHandler(store ExerciseMuscleStore) *ExerciseMuscleHandler {
	return &ExerciseMuscleHandler{exerciseMuscle: store}
}

func (h *ExerciseMuscleHandler) list(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a number")
		return
	}
	userID, ok := userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	muscles, err := h.exerciseMuscle.ListByExercise(r.Context(), userID, id)
	if err != nil {
		slog.Error("list exercise muscles", "exercise_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, muscles)
}

func (h *ExerciseMuscleHandler) replace(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a number")
		return
	}
	var req []exercise.Muscle
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	userID, ok := userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := exercise.ValidateMuscles(req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := h.exerciseMuscle.ReplaceForExercise(r.Context(), userID, id, req); err != nil {
		if errors.Is(err, exercise.ErrNotFound) {
			writeError(w, http.StatusNotFound, "exercise not found")
			return
		}
		if errors.Is(err, exercise.ErrMuscleGroupNotFound) {
			writeError(w, http.StatusUnprocessableEntity, "unknown muscle_group_id")
			return
		}
		slog.Error("replace exercise muscles", "exercise_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	muscles, err := h.exerciseMuscle.ListByExercise(r.Context(), userID, id)
	if err != nil {
		slog.Error("list exercise muscles", "exercise_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, muscles)
}
