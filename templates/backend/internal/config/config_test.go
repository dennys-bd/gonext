package config

import (
	"os"
	"slices"
	"testing"
	"time"
)

const testDatabaseURL = "postgres://app:app@localhost:5432/app?sslmode=disable"

func unsetEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		prev, existed := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unsetenv %s: %v", key, err)
		}
		t.Cleanup(func() {
			if existed {
				if err := os.Setenv(key, prev); err != nil {
					t.Fatalf("setenv %s: %v", key, err)
				}
			}
		})
	}
}

func TestLoad_Defaults(t *testing.T) {
	unsetEnv(t, "ENV", "PORT", "LOG_LEVEL", "LOG_FORMAT", "SHUTDOWN_TIMEOUT", "RATE_LIMIT_RPS", "RATE_LIMIT_BURST", "TRUSTED_PROXIES")
	t.Setenv("DATABASE_URL", testDatabaseURL)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Env != "prod" {
		t.Errorf("expected default Env prod, got %q", cfg.Env)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected default Port 8080, got %d", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default LogLevel info, got %q", cfg.LogLevel)
	}
	if cfg.LogFormat != "pretty" {
		t.Errorf("expected default LogFormat pretty, got %q", cfg.LogFormat)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("expected default ShutdownTimeout 10s, got %s", cfg.ShutdownTimeout)
	}
	if cfg.DatabaseURL != testDatabaseURL {
		t.Errorf("expected DatabaseURL %q, got %q", testDatabaseURL, cfg.DatabaseURL)
	}
	if cfg.RateLimitRPS != 20 {
		t.Errorf("expected default RateLimitRPS 20, got %v", cfg.RateLimitRPS)
	}
	if cfg.RateLimitBurst != 40 {
		t.Errorf("expected default RateLimitBurst 40, got %d", cfg.RateLimitBurst)
	}
	if len(cfg.TrustedProxies) != 0 {
		t.Errorf("expected no default TrustedProxies, got %v", cfg.TrustedProxies)
	}
}

func TestLoad_ValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{"invalid env", map[string]string{"ENV": "staging"}},
		{"invalid log level", map[string]string{"LOG_LEVEL": "verbose"}},
		{"port below range", map[string]string{"PORT": "0"}},
		{"port above range", map[string]string{"PORT": "70000"}},
		{"non-positive shutdown timeout", map[string]string{"SHUTDOWN_TIMEOUT": "0s"}},
		{"invalid log format", map[string]string{"LOG_FORMAT": "xml"}},
		{"non-positive rate limit rps", map[string]string{"RATE_LIMIT_RPS": "0"}},
		{"zero rate limit burst", map[string]string{"RATE_LIMIT_BURST": "0"}},
		{"malformed trusted proxy", map[string]string{"TRUSTED_PROXIES": "nope"}},
		{"one malformed among valid trusted proxies", map[string]string{"TRUSTED_PROXIES": "10.0.0.0/8,nope"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", testDatabaseURL)
			for key, value := range tt.env {
				t.Setenv(key, value)
			}
			if _, err := Load(); err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	unsetEnv(t, "DATABASE_URL")

	if _, err := Load(); err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestLoad_TrustedProxies(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{"empty is none", "", nil},
		{"one", "10.0.0.0/8", []string{"10.0.0.0/8"}},
		{"two", "10.0.0.0/8,192.168.0.0/16", []string{"10.0.0.0/8", "192.168.0.0/16"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", testDatabaseURL)
			t.Setenv("TRUSTED_PROXIES", tt.value)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(cfg.TrustedProxies, tt.want) {
				t.Errorf("TrustedProxies = %v, want %v", cfg.TrustedProxies, tt.want)
			}
		})
	}
}

// stg sits on the restricted side with prod, not the relaxed side with dev/test.
func TestIsRelaxedEnv(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{"dev", true},
		{"test", true},
		{"stg", false},
		{"prod", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			if got := IsRelaxedEnv(tt.env); got != tt.want {
				t.Fatalf("env %q: expected %v, got %v", tt.env, tt.want, got)
			}
		})
	}
}
