package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dennys-bd/gonext/auth"
)

// bearerPrefix is the scheme prefix on an Authorization header.
const bearerPrefix = "Bearer "

// AuthConfig is the middleware's transport policy: where a credential
// is read from. It is policy rather than contract — no provider needs
// it — so it lives here rather than in the published auth module.
type AuthConfig struct {
	// CookieName is the cookie the credential travels in.
	CookieName string
	// AllowBearer additionally accepts an Authorization: Bearer header.
	// Off by default, since the project's only client is a browser.
	AllowBearer bool
}

// ProvideAuthConfig returns the default transport policy.
func ProvideAuthConfig() AuthConfig {
	return AuthConfig{CookieName: auth.DefaultCookieName, AllowBearer: false}
}

// NewAuthMiddleware returns Huma middleware enforcing each operation's
// declared Security and injecting the resolved identity. An operation
// declaring nothing never has its credential read at all.
func NewAuthMiddleware(
	api huma.API,
	resolver auth.Resolver,
	cfg AuthConfig,
	logger *slog.Logger,
) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		rule := auth.RuleFor(ctx.Operation().Security)
		if !rule.Enabled {
			next(ctx)
			return
		}

		token := extractToken(ctx, cfg)
		if token == "" {
			if rule.Optional {
				next(ctx)
				return
			}
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "authentication required")
			return
		}

		identity, err := resolver.Resolve(ctx.Context(), token)
		if err != nil {
			if errors.Is(err, auth.ErrUnauthenticated) {
				if rule.Optional {
					next(ctx)
					return
				}
				_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "authentication required")
				return
			}
			// Provider failure, not an auth outcome: don't send clients to login during an outage.
			logger.ErrorContext(ctx.Context(), "auth: resolving credential", "error", err)
			_ = huma.WriteErr(api, ctx, http.StatusInternalServerError, "internal server error")
			return
		}

		if !satisfies(identity, rule) {
			_ = huma.WriteErr(api, ctx, http.StatusForbidden, "forbidden")
			return
		}

		next(huma.WithValue(ctx, auth.ContextKey, identity))
	}
}

// extractToken pulls the credential off the request. The middleware
// owns this so that a provider never learns about transport.
func extractToken(ctx huma.Context, cfg AuthConfig) string {
	if cookie, err := huma.ReadCookie(ctx, cfg.CookieName); err == nil && cookie != nil && cookie.Value != "" {
		return cookie.Value
	}
	if cfg.AllowBearer {
		if header := ctx.Header("Authorization"); strings.HasPrefix(header, bearerPrefix) {
			return strings.TrimPrefix(header, bearerPrefix)
		}
	}
	return ""
}

// satisfies reports whether identity meets every role and permission (ANDed) the rule requires.
func satisfies(identity auth.Identity, rule auth.Rule) bool {
	for _, role := range rule.Roles {
		if !identity.HasRole(role) {
			return false
		}
	}
	for _, permission := range rule.Permissions {
		if !identity.HasPermission(permission) {
			return false
		}
	}
	return true
}
