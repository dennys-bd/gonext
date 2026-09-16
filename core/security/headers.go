package security

import (
	"slices"

	"github.com/labstack/echo/v4"
)

const cspDefault = "default-src 'none'; frame-ancestors 'none'"
const hstsValue = "max-age=31536000; includeSubDomains"

// HeadersConfig controls what Headers emits.
type HeadersConfig struct {
	// HSTS adds Strict-Transport-Security. Leave it off wherever the
	// server is reached over plain http.
	HSTS bool
	// SkipCSP lists request paths that receive no Content-Security-Policy.
	SkipCSP []string
}

// Headers returns middleware that sets the baseline security headers
// on every response.
func Headers(cfg HeadersConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			h.Set(echo.HeaderXContentTypeOptions, "nosniff")
			h.Set(echo.HeaderXFrameOptions, "DENY")
			h.Set(echo.HeaderReferrerPolicy, "no-referrer")
			if !slices.Contains(cfg.SkipCSP, c.Request().URL.Path) {
				h.Set(echo.HeaderContentSecurityPolicy, cspDefault)
			}
			if cfg.HSTS {
				h.Set(echo.HeaderStrictTransportSecurity, hstsValue)
			}
			return next(c)
		}
	}
}
