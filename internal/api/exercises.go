package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"trainingApp/internal/exercise"
)

const (
	maxBodySize  = 1 << 20 // 1 MB
	defaultLimit = 20
	maxLimit     = 100
)

type createExerciseRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *Handler) createExercise(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	var req createExerciseRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := exercise.Validate(req.Name); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	created, err := h.exercises.Create(r.Context(), exercise.Exercise{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		log.Printf("create exercise: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.Header().Set("Location", "/api/v1/exercises/"+strconv.FormatInt(created.ID, 10))
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) getExercise(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a number")
		return
	}

	e, err := h.exercises.GetByID(r.Context(), id)
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

func (h *Handler) listExercises(w http.ResponseWriter, r *http.Request) {
	limit := defaultLimit
	if s := r.URL.Query().Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > maxLimit {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 100")
			return
		}
		limit = n
	}

	res, err := h.exercises.List(r.Context(), limit)
	if err != nil {
		log.Printf("list exercises: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, res)
}
