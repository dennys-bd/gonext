// Package api hosts the cross-cutting HTTP server bootstrap shared by
// all domains: the single Echo+Huma instance every domain registers onto.
package api

import (
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// NewEcho builds the shared Echo router with its cross-cutting
// middleware. logger is used for the per-request logging middleware.
func NewEcho(logger *slog.Logger) *echo.Echo {
	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(NewLoggingMiddleware(logger))
	return e
}

// NewHumaAPI wraps e in a Huma API that domains register their
// endpoints against.
func NewHumaAPI(e *echo.Echo) huma.API {
	return humaecho.NewV4(e, huma.DefaultConfig("Backend API", "0.1.0"))
}

// NewServer builds the shared Echo router wrapped in a Huma API.
// Domains call huma.Register against the returned huma.API; main.go
// owns starting the returned *echo.Echo.
//
// NewEcho and NewHumaAPI are split out separately so wire can wire
// them as two distinct providers; NewServer stays as the convenience
// entry point for callers (e.g. tests) that want both at once.
func NewServer(logger *slog.Logger) (*echo.Echo, huma.API) {
	e := NewEcho(logger)
	return e, NewHumaAPI(e)
}
