package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"[PROJECT-NAME]/backend/internal/config"
)

func testConfig(env string, burst int) config.Config {
	return config.Config{Env: env, RateLimitRPS: 0.001, RateLimitBurst: burst}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNewEcho_RateLimitsPerClient(t *testing.T) {
	e, err := NewEcho(discardLogger(), testConfig("dev", 2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ok := func(c echo.Context) error { return c.NoContent(http.StatusOK) }
	e.GET("/x", ok)
	e.GET("/healthz", ok)

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("/x hit %d: expected 200, got %d", i, rec.Code)
		}
	}

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("/x third hit: expected 429, got %d", rec.Code)
	}
	if got := rec.Header().Get(echo.HeaderContentType); got != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", got)
	}
	if got := rec.Header().Get(echo.HeaderXContentTypeOptions); got != "nosniff" {
		t.Errorf("expected the 429 to still carry X-Content-Type-Options: nosniff, got %q", got)
	}

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("/healthz hit %d: expected 200, got %d", i, rec.Code)
		}
	}
}

func TestNewEcho_HSTSFollowsEnv(t *testing.T) {
	tests := []struct {
		env      string
		wantHSTS bool
	}{
		{"dev", false},
		{"test", false},
		{"stg", true},
		{"prod", true},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			e, err := NewEcho(discardLogger(), testConfig(tt.env, 40))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			e.GET("/x", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

			got := rec.Header().Get(echo.HeaderStrictTransportSecurity) != ""
			if got != tt.wantHSTS {
				t.Errorf("env %q: HSTS present = %v, want %v", tt.env, got, tt.wantHSTS)
			}
		})
	}
}

func TestNewEcho_RejectsBadTrustedProxy(t *testing.T) {
	_, err := NewEcho(discardLogger(), config.Config{RateLimitRPS: 1, RateLimitBurst: 1, TrustedProxies: []string{"nope"}})
	if err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("expected an error naming %q, got %v", "nope", err)
	}
}
