package security

import (
	"net"
	"net/http"
	"slices"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

const problemJSON = "application/problem+json"

var tooManyRequestsBody = []byte(`{"title":"Too Many Requests","status":429,"detail":"rate limit exceeded"}`)

// RateLimitConfig controls what RateLimit allows.
type RateLimitConfig struct {
	// RPS is the sustained requests per second allowed per client IP.
	RPS float64
	// Burst is how many requests a client may make at once before RPS applies.
	Burst int
	// SkipPaths lists request paths the limiter never counts.
	SkipPaths []string
}

// RateLimit returns middleware that answers 429 once a client exceeds cfg.
// A client is the IPv4 address or the IPv6 /64 the Echo instance's
// IPExtractor resolves, or the TCP peer when none is set. Limits are per
// process; replicas do not share a bucket.
func RateLimit(cfg RateLimitConfig) echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Skipper: func(c echo.Context) bool {
			return slices.Contains(cfg.SkipPaths, c.Request().URL.Path)
		},
		IdentifierExtractor: func(c echo.Context) (string, error) {
			return clientKey(c), nil
		},
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate:  rate.Limit(cfg.RPS),
			Burst: cfg.Burst,
		}),
		DenyHandler: func(c echo.Context, _ string, _ error) error {
			return c.Blob(http.StatusTooManyRequests, problemJSON, tooManyRequestsBody)
		},
	})
}

// clientKey never falls back to Echo's legacy RealIP, which trusts
// X-Forwarded-For from anyone, and masks IPv6 to /64 so one allocation
// cannot mint a fresh bucket per request.
func clientKey(c echo.Context) string {
	extract := c.Echo().IPExtractor
	if extract == nil {
		extract = echo.ExtractIPDirect()
	}
	raw := extract(c.Request())
	ip := net.ParseIP(raw)
	if ip == nil || ip.To4() != nil {
		return raw
	}
	// ponytail: memory store is bounded by time only (3 min expiry); a
	// count-bounded store is Feature Pack D's Redis limiter.
	return ip.Mask(net.CIDRMask(64, 128)).String()
}
