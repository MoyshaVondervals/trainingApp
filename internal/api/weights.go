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
	"trainingApp/internal/weight"
)

const dateLayout = "2006-01-02"

type WeightHandler struct {
	weights WeightStore
}

type WeightStore interface {
	Upsert(ctx context.Context, b weight.BodyWeight) (weight.BodyWeight, error)
	List(ctx context.Context, userID int64, from, to time.Time, limit int) ([]weight.BodyWeight, error)
	Delete(ctx context.Context, userID, id int64) error
}

func NewWeightHandler(store WeightStore) *WeightHandler {
	return &WeightHandler{weights: store}
}

type weightRequest struct {
	WeightKg   float32 `json:"weight_kg"`
	MeasuredOn string  `json:"measured_on"`
	Note       string  `json:"note"`
}

func (h *WeightHandler) create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	var req weightRequest
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

	measuredOn := time.Now().UTC().Truncate(24 * time.Hour)
	if req.MeasuredOn != "" {
		parsed, err := time.Parse(dateLayout, req.MeasuredOn)
		if err != nil {
			writeError(w, http.StatusBadRequest, "measured_on must be a date like 2006-01-02")
			return
		}
		measuredOn = parsed
	}

	bodyWeight := weight.BodyWeight{
		UserID:     userID,
		WeightKg:   req.WeightKg,
		MeasuredOn: measuredOn,
		Note:       req.Note,
	}
	if err := bodyWeight.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	created, err := h.weights.Upsert(r.Context(), bodyWeight)
	if err != nil {
		slog.Error("upsert body weight", "err", err)
		writeError(w, http.StatusInternalServerError, "error saving body weight")
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/api/v1/weights/%d", created.ID))
	writeJSON(w, http.StatusCreated, created)
}

func (h *WeightHandler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	limit, err := parseLimit(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	from, err := parseDateQuery(r, "from")
	if err != nil {
		writeError(w, http.StatusBadRequest, "from must be a date like 2006-01-02")
		return
	}
	to, err := parseDateQuery(r, "to")
	if err != nil {
		writeError(w, http.StatusBadRequest, "to must be a date like 2006-01-02")
		return
	}
	if from.IsZero() {
		from = time.Now().UTC().AddDate(-1, 0, 0)
	}
	if to.IsZero() {
		to = time.Now().UTC()
	}
	if to.Before(from) {
		writeError(w, http.StatusUnprocessableEntity, "to must be later than from")
		return
	}

	found, err := h.weights.List(r.Context(), userID, from, to, limit)
	if err != nil {
		slog.Error("list body weights", "err", err)
		writeError(w, http.StatusInternalServerError, "error listing body weights")
		return
	}
	writeJSON(w, http.StatusOK, found)
}

func (h *WeightHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	userID, ok := userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.weights.Delete(r.Context(), userID, id); err != nil {
		if errors.Is(err, weight.ErrNotFound) {
			writeError(w, http.StatusNotFound, "body weight not found")
			return
		}
		slog.Error("delete body weight", "body_weight_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "error deleting body weight")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseDateQuery(r *http.Request, name string) (time.Time, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return time.Time{}, nil
	}
	return time.Parse(dateLayout, raw)
}
