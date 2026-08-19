package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"trainingApp/internal/exercise"
)

const (
	maxBodySize  = 1 << 20
	defaultLimit = 20
	maxLimit     = 100
)

type ExerciseHandler struct {
	exercises ExerciseStore
}

func NewExerciseHandler(store ExerciseStore) *ExerciseHandler {
	return &ExerciseHandler{exercises: store}

}

type ExerciseStore interface {
	Create(ctx context.Context, e exercise.Exercise) (exercise.Exercise, error)
	GetByID(ctx context.Context, userID, id int64) (exercise.Exercise, error)
	List(ctx context.Context, userID int64, limit int) ([]exercise.Exercise, error)
	UpdateByID(ctx context.Context, userID int64, e exercise.Exercise) (exercise.Exercise, error)
	Delete(ctx context.Context, userID, id int64) error
}

type createExerciseRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *ExerciseHandler) deleteExercise(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
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
	if err := h.exercises.Delete(r.Context(), userID, id); err != nil {
		if errors.Is(err, exercise.ErrNotFound) {
			writeError(w, http.StatusNotFound, "exercise not found")
			return
		}
		log.Printf("delete exercise %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ExerciseHandler) updateExercise(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a number")
		return
	}
	var req createExerciseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	userID, ok := userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	exerciseObj := exercise.Exercise{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
	}
	if err := exerciseObj.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	updated, err := h.exercises.UpdateByID(r.Context(), userID, exerciseObj)
	if err != nil {
		if errors.Is(err, exercise.ErrNotFound) {
			writeError(w, http.StatusNotFound, "exercise not found")
			return
		}
		log.Printf("update exercise %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *ExerciseHandler) createExercise(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	var req createExerciseRequest
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
	exerciseObj := exercise.Exercise{
		Name:        req.Name,
		Description: req.Description,
		UserID:      &userID,
	}
	if err := exerciseObj.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	created, err := h.exercises.Create(r.Context(), exerciseObj)
	if err != nil {
		log.Printf("create exercise: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.Header().Set("Location", "/api/v1/exercises/"+strconv.FormatInt(created.ID, 10))
	writeJSON(w, http.StatusCreated, created)
}

func (h *ExerciseHandler) getExercise(w http.ResponseWriter, r *http.Request) {
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
	e, err := h.exercises.GetByID(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, exercise.ErrNotFound) {
			writeError(w, http.StatusNotFound, "exercise not found")
			return
		}
		log.Printf("get exercise %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, e)
}

func (h *ExerciseHandler) listExercises(w http.ResponseWriter, r *http.Request) {
	limit := defaultLimit
	if s := r.URL.Query().Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > maxLimit {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 100")
			return
		}
		limit = n
	}
	userID, ok := userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	res, err := h.exercises.List(r.Context(), userID, limit)
	if err != nil {
		log.Printf("list exercises: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, res)
}
