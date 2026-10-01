package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
	"trainingApp/internal/api"
	"trainingApp/internal/config"
	"trainingApp/internal/postgres"
)

func newLogger(level slog.Level, format string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}
	if format == "json" {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := newLogger(cfg.SlogLevel(), cfg.LogFormat)
	slog.SetDefault(logger)
	db, err := postgres.Open(context.Background(), cfg.DSN)
	if err != nil {
		return err
	}
	defer db.Close()
	router := api.Router(
		api.NewExerciseHandler(postgres.NewExerciseRepo(db)),
		api.NewUserHandler(postgres.NewUserRepo(db), cfg.Secret(), cfg.JWTTTL),
		api.NewWorkoutsHandler(postgres.NewWorkoutRepo(db)),
		api.NewSetHandler(postgres.NewSetRepo(db)),
		api.NewExerciseMuscleHandler(postgres.NewExerciseMusclesRepo(db)),
		api.NewStatsHandler(postgres.NewStatsRepo(db)),
		api.NewWeightHandler(postgres.NewWeightRepo(db)),
		api.NewPlanHandler(postgres.NewPlanRepo(db)),
	)
	h := api.Recover(logger)(api.RequestLogger(logger)(router))
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           h,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	var pprofSrv *http.Server
	if addr := os.Getenv("PPROF_ADDR"); addr != "" {
		runtime.SetBlockProfileRate(1)
		runtime.SetMutexProfileFraction(1)

		pprofSrv = &http.Server{
			Addr:              addr,
			Handler:           http.DefaultServeMux,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		}
		go func() {
			slog.Info("pprof listening", "addr", addr)
			if err := pprofSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Error("pprof listen", "err", err)
			}
		}()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "err", err)
	}
	if pprofSrv != nil {
		if err := pprofSrv.Shutdown(shutdownCtx); err != nil {
			slog.Error("pprof shutdown", "err", err)
		}
	}

	return nil
}
