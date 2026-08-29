package config

import (
	"os"
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
	unsetEnv(t, "ENV", "PORT", "LOG_LEVEL", "LOG_FORMAT", "SHUTDOWN_TIMEOUT")
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
