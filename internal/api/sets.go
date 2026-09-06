package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"trainingApp/internal/set"
)

const maxSetsPerWorkout = 900

type SetHandler struct {
	sets SetsStore
}
type SetsStore interface {
	Create(ctx context.Context, userID int64, s set.Set) (set.Set, error)
	Update(ctx context.Context, userID int64, s set.Set) (set.Set, error)
	Delete(ctx context.Context, userID int64, s set.Set) error
	GetById(ctx context.Context, userID, id int64) (set.Set, error)
	ListByWorkout(ctx context.Context, userID, workoutID int64, limit int) ([]set.Set, error)
	LastPerformance(ctx context.Context, userID, exerciseID, excludeWorkoutID int64, planID *int64) (set.LastPerformance, error)
}

func NewSetHandler(s SetsStore) *SetHandler {
	return &SetHandler{sets: s}
}

type createSetRequest struct {
	ExerciseID int64   `json:"exercise_id"`
	WorkoutID  int64   `json:"workout_id"`
	SetNumber  int64   `json:"set_number"`
	Reps       int     `json:"reps"`
	Weight     float32 `json:"weight"`
}

type updateSetRequest struct {
	SetNumber int64   `json:"set_number"`
	Reps      int     `json:"reps"`
	Weight    float32 `json:"weight"`
}

func (h *SetHandler) create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	var req createSetRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userId, ok := userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	setObj := set.Set{
		ExerciseID: req.ExerciseID,
		WorkoutID:  req.WorkoutID,
		SetNumber:  req.SetNumber,
		Reps:       req.Reps,
		Weight:     req.Weight,
	}
	if err := setObj.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	created, err := h.sets.Create(r.Context(), userId, setObj)
	if err != nil {
		if errors.Is(err, set.ErrNotFound) {
			writeError(w, http.StatusNotFound, "workout not found")
			return
		}
		if errors.Is(err, set.ErrAlreadyExists) {
			writeError(w, http.StatusConflict, "set with this number already exists for the exercise")
			return
		}
		slog.Error("create set", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/api/v1/sets/%d", created.ID))
	writeJSON(w, http.StatusCreated, created)
}

func (h *SetHandler) update(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req updateSetRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userId, ok := userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	setObj := set.Set{
		ID:        id,
		SetNumber: req.SetNumber,
		Reps:      req.Reps,
		Weight:    req.Weight,
	}
	if err := setObj.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	updated, err := h.sets.Update(r.Context(), userId, setObj)
	if err != nil {
		if errors.Is(err, set.ErrNotFound) {
			writeError(w, http.StatusNotFound, set.ErrNotFound.Error())
			return
		}
		if errors.Is(err, set.ErrAlreadyExists) {
			writeError(w, http.StatusConflict, "set with this number already exists for the exercise")
			return
		}
		slog.Error("update set", "set_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *SetHandler) delete(w http.ResponseWriter, r *http.Request) {
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
	setObj := set.Set{
		ID: id,
	}
	if err := h.sets.Delete(r.Context(), userId, setObj); err != nil {
		if errors.Is(err, set.ErrNotFound) {
			writeError(w, http.StatusNotFound, set.ErrNotFound.Error())
			return
		}
		slog.Error("set handler", "set_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (h *SetHandler) getById(w http.ResponseWriter, r *http.Request) {
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

	s, err := h.sets.GetById(r.Context(), userId, id)
	if err != nil {
		if errors.Is(err, set.ErrNotFound) {
			writeError(w, http.StatusNotFound, set.ErrNotFound.Error())
			return
		}
		slog.Error("set handler", "set_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, s)

}

func (h *SetHandler) listByWorkout(w http.ResponseWriter, r *http.Request) {
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
	s, err := h.sets.ListByWorkout(r.Context(), userId, id, maxSetsPerWorkout)
	if err != nil {
		if errors.Is(err, set.ErrNotFound) {
			writeError(w, http.StatusNotFound, set.ErrNotFound.Error())
			return
		}
		slog.Error("set handler", "set_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *SetHandler) lastByExercise(w http.ResponseWriter, r *http.Request) {
	exerciseID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	userID, ok := userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var excludeWorkoutID int64
	if raw := r.URL.Query().Get("exclude_workout"); raw != "" {
		excludeWorkoutID, err = strconv.ParseInt(raw, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "exclude_workout must be a number")
			return
		}
	}

	var planID *int64
	if raw := r.URL.Query().Get("plan_id"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "plan_id must be a number")
			return
		}
		planID = &parsed
	}

	last, err := h.sets.LastPerformance(r.Context(), userID, exerciseID, excludeWorkoutID, planID)
	if err != nil {
		if errors.Is(err, set.ErrNotFound) {
			writeError(w, http.StatusNotFound, "no previous sets for this exercise")
			return
		}
		slog.Error("last performance", "exercise_id", exerciseID, "err", err)
		writeError(w, http.StatusInternalServerError, "error getting last sets")
		return
	}
	writeJSON(w, http.StatusOK, last)
}
