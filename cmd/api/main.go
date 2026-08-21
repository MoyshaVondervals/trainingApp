package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"trainingApp/internal/api"
	"trainingApp/internal/config"
	"trainingApp/internal/postgres"

	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := postgres.Open(context.Background(), cfg.DSN)
	if err != nil {
		return err
	}
	defer db.Close()
	h := api.Router(
		api.NewExerciseHandler(postgres.NewExerciseRepo(db)),
		api.NewUserHandler(postgres.NewUserRepo(db), cfg.JWTSecret, cfg.JWTTTL),
		api.NewWorkoutsHandler(postgres.NewWorkoutRepo(db)),
		api.NewSetHandler(postgres.NewSetRepo(db)),
	)
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
			log.Fatalf("listen: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}

	return nil
}
