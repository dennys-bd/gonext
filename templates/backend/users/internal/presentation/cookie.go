package presentation

import (
	"net/http"
	"time"

	"[PROJECT-NAME]/backend/users/internal/application"
)

// SessionCookieName is the cookie the session token travels in.
const SessionCookieName = "session"

// CookieOptions carries the transport-level session cookie policy.
// It lives here rather than in the domain or application layers:
// those only ever handle a raw token string, so swapping the delivery
// mechanism (a bearer header, a JWT for mobile) touches nothing else.
type CookieOptions struct {
	// Secure marks the cookie HTTPS-only. It is off only in a relaxed
	// environment, where the server is reached over plain http and a
	// Secure cookie would be silently dropped by the browser.
	Secure bool
}

// NewCookieOptions derives the cookie policy from the runtime
// environment. It gates on application.IsRelaxedEnv — the same
// predicate the use cases gate on — so cookie policy and use-case
// policy cannot disagree about what counts as a local environment.
func NewCookieOptions(env string) CookieOptions {
	return CookieOptions{Secure: !application.IsRelaxedEnv(env)}
}

// sessionCookie builds the Set-Cookie value that stores token.
func (o CookieOptions) sessionCookie(token string, expiresAt time.Time) http.Cookie {
	return http.Cookie{
		Name:     SessionCookieName,
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
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   o.Secure,
		SameSite: http.SameSiteLaxMode,
	}
}
