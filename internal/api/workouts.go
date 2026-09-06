package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"
	"trainingApp/internal/workout"
)

type WorkoutsHandler struct {
	workouts WorkoutStore
}
type WorkoutStore interface {
	Create(ctx context.Context, w workout.Workout) (workout.Workout, error)
	GetByID(ctx context.Context, userID, id int64) (workout.Workout, error)
	List(ctx context.Context, userID int64, limit int) ([]workout.Workout, error)
	UpdateNoteByID(ctx context.Context, userID int64, e workout.Workout) (workout.Workout, error)
	FinishTraining(ctx context.Context, userID, id int64) (workout.Workout, error)
	Delete(ctx context.Context, userID, id int64) error
}

func NewWorkoutsHandler(store WorkoutStore) *WorkoutsHandler {
	return &WorkoutsHandler{workouts: store}
}

type workoutRequest struct {
	StartedAt time.Time `json:"started_at"`
	Note      string    `json:"note"`
	PlanID    *int64    `json:"plan_id"`
}

func (h *WorkoutsHandler) startTraining(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	var req workoutRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	userId, ok := userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	startedAt := req.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now()
	}
	workoutObj := workout.Workout{
		UserID:    userId,
		StartedAt: startedAt,
		Note:      req.Note,
		PlanID:    req.PlanID,
	}
	if err := workoutObj.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	created, err := h.workouts.Create(r.Context(), workoutObj)
	if err != nil {
		if errors.Is(err, workout.ErrPlanNotFound) {
			writeError(w, http.StatusNotFound, "plan not found")
			return
		}
		slog.Error("create workout", "err", err)
		writeError(w, http.StatusInternalServerError, "error creating workout")
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/api/v1/workouts/%d", created.ID))
	writeJSON(w, http.StatusCreated, created)
}

func (h *WorkoutsHandler) getTraining(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	userId, ok := userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	e, err := h.workouts.GetByID(r.Context(), userId, id)
	if err != nil {
		if errors.Is(err, workout.ErrNotFound) {
			writeError(w, http.StatusNotFound, "workout not found")
			return
		}
		slog.Error("get workout", "workout_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "error getting workout")
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *WorkoutsHandler) listWorkout(w http.ResponseWriter, r *http.Request) {
	limit := defaultLimit
	if s := r.URL.Query().Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > maxLimit {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 100")
			return
		}
		limit = n
	}
	userId, ok := userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	e, err := h.workouts.List(r.Context(), userId, limit)
	if err != nil {
		slog.Error("list workouts", "err", err)
		writeError(w, http.StatusInternalServerError, "error getting workouts")
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *WorkoutsHandler) updateNote(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req workoutRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	userId, ok := userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	workoutObj := workout.Workout{
		ID:   id,
		Note: req.Note,
	}
	if err := workoutObj.ValidateNote(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	updated, err := h.workouts.UpdateNoteByID(r.Context(), userId, workoutObj)
	if err != nil {
		if errors.Is(err, workout.ErrNotFound) {
			writeError(w, http.StatusNotFound, "workout not found")
			return
		}
		slog.Error("update workout", "workout_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "error updating workout")
		return

	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *WorkoutsHandler) finishTraining(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	userId, ok := userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	finished, err := h.workouts.FinishTraining(r.Context(), userId, id)
	if err != nil {
		if errors.Is(err, workout.ErrNotFound) {
			writeError(w, http.StatusNotFound, "workout not found")
			return
		}
		slog.Error("finish workout", "workout_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "error finishing workout")
		return
	}
	writeJSON(w, http.StatusOK, finished)
}

func (h *WorkoutsHandler) deleteTraining(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	userId, ok := userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.workouts.Delete(r.Context(), userId, id); err != nil {
		if errors.Is(err, workout.ErrNotFound) {
			writeError(w, http.StatusNotFound, "workout not found")
			return
		}
		slog.Error("delete workout", "workout_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "error deleting workout")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)

}
