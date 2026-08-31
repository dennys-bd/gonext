// Package api hosts the cross-cutting HTTP server bootstrap shared by
// all domains: the single Echo+Huma instance every domain registers onto.
package api

import (
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
	"github.com/dennys-bd/gonext/auth"
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
// endpoints against, publishing the session security scheme and
// installing the auth middleware that enforces it.
func NewHumaAPI(e *echo.Echo, resolver auth.Resolver, cfg AuthConfig, logger *slog.Logger) huma.API {
	humaConfig := huma.DefaultConfig("Backend API", "0.1.0")
	humaConfig.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		auth.SchemeName: {
			Type:        "apiKey",
			In:          "cookie",
			Name:        auth.DefaultCookieName,
			Description: "Opaque session issued by POST /users/login.",
		},
	}

	api := humaecho.NewV4(e, humaConfig)
	api.UseMiddleware(NewAuthMiddleware(api, resolver, cfg, logger))
	return api
}

// NewServer builds the shared Echo router wrapped in a Huma API.
// Domains call httpx.Register against the returned huma.API; main.go
// owns starting the returned *echo.Echo.
//
// NewEcho and NewHumaAPI stay split so wire can treat them as two
// providers; NewServer is the convenience entry point for callers
// (e.g. tests) that want both at once.
func NewServer(logger *slog.Logger, resolver auth.Resolver, cfg AuthConfig) (*echo.Echo, huma.API) {
	e := NewEcho(logger)
	return e, NewHumaAPI(e, resolver, cfg, logger)
}
