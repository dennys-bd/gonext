package api

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
)

// NewLoggingMiddleware returns Echo middleware that logs one line per request
// via logger, at Error for a 5xx status or handler error, Info otherwise.
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
