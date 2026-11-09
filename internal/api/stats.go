package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
	"trainingApp/internal/stats"

	"golang.org/x/sync/errgroup"
)

type StatsHandler struct {
	stats StatsStore
	cache DashboardCache
}

type DashboardCache interface {
	Get(context.Context, int64, string, string) (stats.Dashboard, bool, error)
	Set(context.Context, int64, string, string, stats.Dashboard) error
}

type StatsStore interface {
	MuscleLoad(ctx context.Context, userID int64, p stats.Period) ([]stats.GroupRoleLoad, error)
	Records(ctx context.Context, userID int64, limit int) ([]stats.Record, error)
	Summary(ctx context.Context, userID int64, p stats.Period) (stats.Summary, error)
}

func NewStatsHandler(store StatsStore, cache DashboardCache) *StatsHandler {
	return &StatsHandler{stats: store, cache: cache}
}

func (h *StatsHandler) dashboard(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	from, err := parseTimeQuery(r, "from")
	if err != nil {
		writeError(w, http.StatusBadRequest, "from must be a valid date")
		return
	}
	to, err := parseTimeQuery(r, "to")
	if err != nil {
		writeError(w, http.StatusBadRequest, "to must be a valid date")
		return
	}
	period, err := stats.NewPeriod(from, to)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	fromRaw := r.URL.Query().Get("from")
	toRaw := r.URL.Query().Get("to")
	if h.cache != nil {
		cached, found, err := h.cache.Get(r.Context(), userID, fromRaw, toRaw)
		if err != nil {
			slog.Warn("read stats cache", "user_id", userID, "err", err)
		} else if found {
			writeJSON(w, http.StatusOK, cached)
			return
		}
	}

	var (
		summary stats.Summary
		muscles []stats.GroupRoleLoad
		records []stats.Record
	)

	g, ctx := errgroup.WithContext(r.Context())
	g.Go(func() error {
		var err error
		summary, err = h.stats.Summary(ctx, userID, period)
		return err
	})
	g.Go(func() error {
		var err error
		muscles, err = h.stats.MuscleLoad(ctx, userID, period)
		return err
	})
	g.Go(func() error {
		var err error
		records, err = h.stats.Records(ctx, userID, stats.MaxRecords())
		return err
	})

	if err := g.Wait(); err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		slog.Error("stats dashboard", "user_id", userID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	result := stats.Dashboard{
		Period:  period,
		Summary: summary,
		Muscles: stats.AggregateByGroup(muscles),
		Records: records,
	}
	if h.cache != nil {
		if err := h.cache.Set(r.Context(), userID, fromRaw, toRaw, result); err != nil {
			slog.Warn("write stats cache", "user_id", userID, "err", err)
		}
	}
	writeJSON(w, http.StatusOK, result)
}

func parseTimeQuery(r *http.Request, name string) (time.Time, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, raw)
}
