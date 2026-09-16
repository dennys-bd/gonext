// Package api hosts the cross-cutting HTTP server bootstrap shared by
// all domains: the single Echo+Huma instance every domain registers onto.
package api

import (
	"fmt"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
	"github.com/dennys-bd/gonext/auth"
	"github.com/dennys-bd/gonext/core/security"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"golden-app/backend/internal/config"
)

// NewEcho builds the shared Echo router with its cross-cutting
// middleware and client-IP policy from cfg. It errors on a
// TrustedProxies entry that is not a CIDR.
func NewEcho(logger *slog.Logger, cfg config.Config) (*echo.Echo, error) {
	extractor, err := security.IPExtractor(cfg.TrustedProxies)
	if err != nil {
		return nil, fmt.Errorf("configuring client ip extraction: %w", err)
	}
	e := echo.New()
	e.IPExtractor = extractor
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(NewLoggingMiddleware(logger))
	e.Use(security.Headers(security.HeadersConfig{
		HSTS:    !config.IsRelaxedEnv(cfg.Env),
		SkipCSP: []string{"/docs"},
	}))
	e.Use(security.RateLimit(security.RateLimitConfig{
		RPS:       cfg.RateLimitRPS,
		Burst:     cfg.RateLimitBurst,
		SkipPaths: []string{"/healthz", "/readyz"},
	}))
	return e, nil
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

// NewServer builds the shared Echo router wrapped in a Huma API. Domains
// call httpx.Register against the returned huma.API; main.go owns starting
// the returned *echo.Echo.
func NewServer(logger *slog.Logger, resolver auth.Resolver, authCfg AuthConfig, cfg config.Config) (*echo.Echo, huma.API, error) {
	e, err := NewEcho(logger, cfg)
	if err != nil {
		return nil, nil, err
	}
	return e, NewHumaAPI(e, resolver, authCfg, logger), nil
}
