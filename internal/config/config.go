package config

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

const envFile = ".env"

type Config struct {
	Env               string        `env:"APP_ENV" env-default:"local"`
	Port              string        `env:"PORT" env-default:"8080"`
	DSN               string        `env:"DATABASE_URL" env-required:"true"`
	ReadHeaderTimeout time.Duration `env:"READ_HEADER_TIMEOUT" env-default:"5s"`
	ReadTimeout       time.Duration `env:"READ_TIMEOUT" env-default:"10s"`
	WriteTimeout      time.Duration `env:"WRITE_TIMEOUT" env-default:"15s"`
	IdleTimeout       time.Duration `env:"IDLE_TIMEOUT" env-default:"60s"`
	JWTSecret         string        `env:"JWT_SECRET" env-required:"true"`
	JWTTTL            time.Duration `env:"JWT_TTL" env-default:"30m"`
	LogLevel          string        `env:"LOG_LEVEL" env-default:"info"`
	LogFormat         string        `env:"LOG_FORMAT" env-default:"text"`
}

func Load() (Config, error) {
	var cfg Config

	err := cleanenv.ReadConfig(envFile, &cfg)
	if errors.Is(err, fs.ErrNotExist) {
		err = cleanenv.ReadEnv(&cfg)
	}
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	switch c.LogFormat {
	case "text", "json":
	default:
		return fmt.Errorf("LOG_FORMAT must be text or json, got %q", c.LogFormat)
	}
	if _, err := parseLevel(c.LogLevel); err != nil {
		return err
	}
	return nil
}

func (c Config) SlogLevel() slog.Level {
	level, err := parseLevel(c.LogLevel)
	if err != nil {
		return slog.LevelInfo
	}
	return level
}

func (c Config) Secret() []byte {
	return []byte(c.JWTSecret)
}

func parseLevel(s string) (slog.Level, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(strings.ToUpper(s))); err != nil {
		return 0, fmt.Errorf("LOG_LEVEL must be debug, info, warn or error, got %q", s)
	}
	return level, nil
}
