package security

import (
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

// RateLimit returns middleware that answers 429 once a client IP exceeds
// cfg. Limits are per process; replicas do not share a bucket.
func RateLimit(cfg RateLimitConfig) echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Skipper: func(c echo.Context) bool {
			return slices.Contains(cfg.SkipPaths, c.Request().URL.Path)
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
