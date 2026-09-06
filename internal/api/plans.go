package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"trainingApp/internal/plan"
)

type PlanHandler struct {
	plans PlanStore
}

type PlanStore interface {
	Create(ctx context.Context, p plan.Plan) (plan.Plan, error)
	Update(ctx context.Context, userID int64, p plan.Plan) (plan.Plan, error)
	List(ctx context.Context, userID int64, limit int) ([]plan.Plan, error)
	GetByID(ctx context.Context, userID, id int64) (plan.Plan, error)
	Delete(ctx context.Context, userID, id int64) error
}

func NewPlanHandler(store PlanStore) *PlanHandler {
	return &PlanHandler{plans: store}
}

type planRequest struct {
	Name      string      `json:"name"`
	Note      string      `json:"note"`
	Exercises []plan.Item `json:"exercises"`
}

func (h *PlanHandler) create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	var req planRequest
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

	planObj := plan.Plan{
		UserID:    userID,
		Name:      req.Name,
		Note:      req.Note,
		Exercises: req.Exercises,
	}
	if err := planObj.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	created, err := h.plans.Create(r.Context(), planObj)
	if err != nil {
		switch {
		case errors.Is(err, plan.ErrAlreadyExists):
			writeError(w, http.StatusConflict, "plan with this name already exists")
		case errors.Is(err, plan.ErrNotFound):
			writeError(w, http.StatusNotFound, "exercise not found")
		default:
			slog.Error("create plan", "err", err)
			writeError(w, http.StatusInternalServerError, "error creating plan")
		}
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/api/v1/plans/%d", created.ID))
	writeJSON(w, http.StatusCreated, created)
}

func (h *PlanHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	var req planRequest
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

	planObj := plan.Plan{
		ID:        id,
		UserID:    userID,
		Name:      req.Name,
		Note:      req.Note,
		Exercises: req.Exercises,
	}
	if err := planObj.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	updated, err := h.plans.Update(r.Context(), userID, planObj)
	if err != nil {
		switch {
		case errors.Is(err, plan.ErrNotFound):
			writeError(w, http.StatusNotFound, "plan or exercise not found")
		case errors.Is(err, plan.ErrAlreadyExists):
			writeError(w, http.StatusConflict, "plan with this name already exists")
		default:
			slog.Error("update plan", "plan_id", id, "err", err)
			writeError(w, http.StatusInternalServerError, "error updating plan")
		}
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *PlanHandler) list(w http.ResponseWriter, r *http.Request) {
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
	plans, err := h.plans.List(r.Context(), userID, limit)
	if err != nil {
		slog.Error("list plans", "err", err)
		writeError(w, http.StatusInternalServerError, "error listing plans")
		return
	}
	writeJSON(w, http.StatusOK, plans)
}

func (h *PlanHandler) get(w http.ResponseWriter, r *http.Request) {
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
	found, err := h.plans.GetByID(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, plan.ErrNotFound) {
			writeError(w, http.StatusNotFound, "plan not found")
			return
		}
		slog.Error("get plan", "plan_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "error getting plan")
		return
	}
	writeJSON(w, http.StatusOK, found)
}

func (h *PlanHandler) delete(w http.ResponseWriter, r *http.Request) {
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
	if err := h.plans.Delete(r.Context(), userID, id); err != nil {
		if errors.Is(err, plan.ErrNotFound) {
			writeError(w, http.StatusNotFound, "plan not found")
			return
		}
		slog.Error("delete plan", "plan_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "error deleting plan")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
