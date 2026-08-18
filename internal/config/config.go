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
	JWTSecret         []byte
	JWTTTL            time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Env:               os.Getenv("APP_ENV"),
		Port:              os.Getenv("PORT"),
		DSN:               os.Getenv("DATABASE_URL"),
		ReadHeaderTimeout: time.Second * 5,
		ReadTimeout:       time.Second * 10,
		WriteTimeout:      time.Second * 15,
		IdleTimeout:       time.Second * 60,
		JWTSecret:         []byte(os.Getenv("JWT_SECRET")),
		JWTTTL:            envDuration("JWT_TTL", 30*time.Minute),
	}

	return cfg, nil

}

func envDuration(name string, fallback time.Duration) time.Duration {
	v := os.Getenv(name)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
