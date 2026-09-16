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
	Env             string        `env:"ENV" envDefault:"prod" validate:"oneof=dev test stg prod"`
	Port            int           `env:"PORT" envDefault:"8080" validate:"min=1,max=65535"`
	LogLevel        string        `env:"LOG_LEVEL" envDefault:"info" validate:"oneof=debug info warn error"`
	LogFormat       string        `env:"LOG_FORMAT" envDefault:"pretty" validate:"oneof=json pretty"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s" validate:"gt=0"`
	DatabaseURL     string        `env:"DATABASE_URL" validate:"required"`
	RateLimitRPS    float64       `env:"RATE_LIMIT_RPS" envDefault:"20" validate:"gt=0"`
	RateLimitBurst  int           `env:"RATE_LIMIT_BURST" envDefault:"40" validate:"min=1"`
	TrustedProxies  []string      `env:"TRUSTED_PROXIES" envSeparator:"," validate:"dive,cidr"`
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

// IsRelaxedEnv reports whether env relaxes safety for local development: raw
// tokens are returned, unconfirmed logins are allowed, the cookie drops Secure
// and HSTS is off. Everything else, including stg, is restricted; both the
// application and HTTP layers gate on this.
func IsRelaxedEnv(env string) bool {
	return env == "dev" || env == "test"
}
