package api

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
)

// NewLoggingMiddleware returns Echo middleware that logs one line per
// request via logger: method, path, status, duration, and the
// request ID set by middleware.RequestID(). It logs at Info for
// 2xx/3xx/4xx responses and Error for 5xx responses or handler errors.
func NewLoggingMiddleware(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)

			status := c.Response().Status
			attrs := []any{
				"method", c.Request().Method,
				"path", c.Path(),
				"status", status,
				"duration", time.Since(start).String(),
				"request_id", c.Response().Header().Get(echo.HeaderXRequestID),
			}

			if status >= 500 || err != nil {
				logger.Error("request", attrs...)
			} else {
				logger.Info("request", attrs...)
			}

			return err
		}
	}
}
