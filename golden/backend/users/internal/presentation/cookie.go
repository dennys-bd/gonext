package presentation

import (
	"net/http"
	"time"

	"github.com/dennys-bd/gonext/auth"

	"golden-app/backend/internal/config"
)

// CookieOptions carries the transport-level session cookie policy.
type CookieOptions struct {
	// Secure marks the cookie HTTPS-only. It is off only in a relaxed
	// environment, where the server is reached over plain http and a
	// Secure cookie would be silently dropped by the browser.
	Secure bool
}

// NewCookieOptions derives the cookie policy from env, gating on
// config.IsRelaxedEnv so cookie and use-case policy cannot disagree.
func NewCookieOptions(env string) CookieOptions {
	return CookieOptions{Secure: !config.IsRelaxedEnv(env)}
}

// sessionCookie builds the Set-Cookie value that stores token.
func (o CookieOptions) sessionCookie(token string, expiresAt time.Time) http.Cookie {
	return http.Cookie{
		Name:     auth.DefaultCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   o.Secure,
		SameSite: http.SameSiteLaxMode,
	}
}

// clearedSessionCookie builds the Set-Cookie value that removes the
// session cookie from the client.
func (o CookieOptions) clearedSessionCookie() http.Cookie {
	return http.Cookie{
		Name:     auth.DefaultCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   o.Secure,
		SameSite: http.SameSiteLaxMode,
	}
}
