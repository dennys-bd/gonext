// Package config loads and validates the backend's runtime
// configuration from environment variables.
package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
)

// Config holds the backend's runtime settings, populated from
// environment variables by Load.
type Config struct {
	Port            int           `env:"PORT" envDefault:"8080" validate:"min=1,max=65535"`
	LogLevel        string        `env:"LOG_LEVEL" envDefault:"info" validate:"oneof=debug info warn error"`
	LogFormat       string        `env:"LOG_FORMAT" envDefault:"pretty" validate:"oneof=json pretty"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s" validate:"gt=0"`
	DatabaseURL     string        `env:"DATABASE_URL" validate:"required"`
}

// Load parses Config from environment variables and validates it.
func Load() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("parsing config: %w", err)
	}
	if err := validator.New().Struct(cfg); err != nil {
		return Config{}, fmt.Errorf("validating config: %w", err)
	}
	return cfg, nil
}
