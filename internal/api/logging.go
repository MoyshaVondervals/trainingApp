package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

type userHolder struct {
	userID int64
	set    bool
}

const userHolderKey ctxKey = 1

func rememberUserID(ctx context.Context, userID int64) {
	if h, ok := ctx.Value(userHolderKey).(*userHolder); ok {
		h.userID = userID
		h.set = true
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusRecorder) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusRecorder) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			holder := &userHolder{}
			r = r.WithContext(context.WithValue(r.Context(), userHolderKey, holder))

			next.ServeHTTP(rec, r)

			attrs := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"dur_ms", time.Since(start).Milliseconds(),
				"bytes", rec.bytes,
			}
			if holder.set {
				attrs = append(attrs, "user_id", holder.userID)
			}

			switch {
			case rec.status >= 500:
				logger.Error("request", attrs...)
			case rec.status >= 400:
				logger.Warn("request", attrs...)
			default:
				logger.Info("request", attrs...)
			}
		})
	}
}
