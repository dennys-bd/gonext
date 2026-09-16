package security_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dennys-bd/gonext/core/security"
	"github.com/labstack/echo/v4"
)

// limited returns an Echo instance with RateLimit(cfg) mounted and two
// routes, /x and /healthz, both returning 200.
func limited(t *testing.T, cfg security.RateLimitConfig) *echo.Echo {
	t.Helper()
	e := echo.New()
	e.Use(security.RateLimit(cfg))
	ok := func(c echo.Context) error { return c.NoContent(http.StatusOK) }
	e.GET("/x", ok)
	e.GET("/healthz", ok)
	return e
}

// hit issues one GET path request from remoteAddr and returns the
// recorded response. A bare echo.New() has no IPExtractor, so the
// limiter keys on RemoteAddr.
func hit(e *echo.Echo, path, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestRateLimit_BurstThen429(t *testing.T) {
	e := limited(t, security.RateLimitConfig{RPS: 0.001, Burst: 3})

	for i := 0; i < 3; i++ {
		rec := hit(e, "/x", "10.0.0.1:1")
		if rec.Code != http.StatusOK {
			t.Fatalf("hit %d: expected 200, got %d", i, rec.Code)
		}
	}

	rec := hit(e, "/x", "10.0.0.1:1")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
	if got := rec.Header().Get(echo.HeaderContentType); got != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", got)
	}
	want := `{"title":"Too Many Requests","status":429,"detail":"rate limit exceeded"}`
	if got := rec.Body.String(); got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestRateLimit_SkipPathsNeverCount(t *testing.T) {
	e := limited(t, security.RateLimitConfig{RPS: 0.001, Burst: 1, SkipPaths: []string{"/healthz"}})

	for i := 0; i < 5; i++ {
		rec := hit(e, "/healthz", "10.0.0.1:1")
		if rec.Code != http.StatusOK {
			t.Fatalf("healthz hit %d: expected 200, got %d", i, rec.Code)
		}
	}

	if rec := hit(e, "/x", "10.0.0.1:1"); rec.Code != http.StatusOK {
		t.Fatalf("first /x hit: expected 200, got %d", rec.Code)
	}
	if rec := hit(e, "/x", "10.0.0.1:1"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second /x hit: expected 429, got %d", rec.Code)
	}
}

func TestRateLimit_BucketsArePerClientIP(t *testing.T) {
	e := limited(t, security.RateLimitConfig{RPS: 0.001, Burst: 1})

	if rec := hit(e, "/x", "10.0.0.1:1"); rec.Code != http.StatusOK {
		t.Fatalf("client 1 first hit: expected 200, got %d", rec.Code)
	}
	if rec := hit(e, "/x", "10.0.0.1:1"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("client 1 second hit: expected 429, got %d", rec.Code)
	}
	if rec := hit(e, "/x", "10.0.0.2:1"); rec.Code != http.StatusOK {
		t.Fatalf("client 2 first hit: expected 200, got %d", rec.Code)
	}
}

func TestRateLimit_IPv6SharesBucketPer64(t *testing.T) {
	e := limited(t, security.RateLimitConfig{RPS: 0.001, Burst: 1})

	if rec := hit(e, "/x", "[2001:db8:1:2::1]:1"); rec.Code != http.StatusOK {
		t.Fatalf("first address in the /64: expected 200, got %d", rec.Code)
	}
	if rec := hit(e, "/x", "[2001:db8:1:2:ffff::9]:1"); rec.Code != http.StatusTooManyRequests {
		t.Errorf("second address in the same /64: expected 429, got %d", rec.Code)
	}
	if rec := hit(e, "/x", "[2001:db8:1:3::1]:1"); rec.Code != http.StatusOK {
		t.Errorf("address in a different /64: expected 200, got %d", rec.Code)
	}
}

func TestRateLimit_IgnoresForwardingHeadersWithoutExtractor(t *testing.T) {
	e := limited(t, security.RateLimitConfig{RPS: 0.001, Burst: 1})

	for i, spoofed := range []string{"1.1.1.1", "2.2.2.2"} {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.RemoteAddr = "10.0.0.1:1"
		req.Header.Set(echo.HeaderXForwardedFor, spoofed)
		req.Header.Set(echo.HeaderXRealIP, spoofed)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		want := http.StatusOK
		if i == 1 {
			want = http.StatusTooManyRequests
		}
		if rec.Code != want {
			t.Errorf("hit %d with spoofed %s: expected %d, got %d", i, spoofed, want, rec.Code)
		}
	}
}
