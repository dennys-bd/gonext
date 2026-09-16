package security_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dennys-bd/gonext/core/security"
	"github.com/labstack/echo/v4"
)

// serve runs a single GET request against a fresh Echo instance with mw
// mounted, returning the recorded response.
func serve(t *testing.T, mw echo.MiddlewareFunc, path string) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	e.Use(mw)
	e.GET(path, func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestHeaders(t *testing.T) {
	tests := []struct {
		name string
		cfg  security.HeadersConfig
		path string
		want map[string]string
	}{
		{
			name: "fixed headers on any path",
			cfg:  security.HeadersConfig{},
			path: "/x",
			want: map[string]string{
				"X-Content-Type-Options":    "nosniff",
				"X-Frame-Options":           "DENY",
				"Referrer-Policy":           "no-referrer",
				"Content-Security-Policy":   "default-src 'none'; frame-ancestors 'none'",
				"Strict-Transport-Security": "",
			},
		},
		{
			name: "skipped path has no CSP but keeps the rest",
			cfg:  security.HeadersConfig{SkipCSP: []string{"/docs"}},
			path: "/docs",
			want: map[string]string{
				"Content-Security-Policy": "",
				"X-Content-Type-Options":  "nosniff",
			},
		},
		{
			name: "skip is exact",
			cfg:  security.HeadersConfig{SkipCSP: []string{"/docs"}},
			path: "/docs/x",
			want: map[string]string{
				"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
			},
		},
		{
			name: "hsts on",
			cfg:  security.HeadersConfig{HSTS: true},
			path: "/x",
			want: map[string]string{
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(t, security.Headers(tt.cfg), tt.path)
			for header, want := range tt.want {
				if got := rec.Header().Get(header); got != want {
					t.Errorf("header %q = %q, want %q", header, got, want)
				}
			}
		})
	}
}
