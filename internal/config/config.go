package config

import (
	"os"
	"time"
)

type Config struct {
	Env               string
	Port              string
	DSN               string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Env:               env("ENV", "local"),
		Port:              env("PORT", "8080"),
		DSN:               env("DATABASE_URL", "postgres://user:secret@localhost:5433/training?sslmode=disable"),
		ReadTimeout:       time.Second * 10,
		ReadHeaderTimeout: time.Second * 5,
		WriteTimeout:      time.Second * 15,
		IdleTimeout:       time.Second * 60,
	}

	return cfg, nil

}

func env(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
