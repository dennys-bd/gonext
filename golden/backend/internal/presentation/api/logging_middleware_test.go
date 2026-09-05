package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dennys-bd/gonext/auth"
	"github.com/labstack/echo/v4"
)

// noopResolver stands in for an identity provider on the endpoints
// these tests exercise. /healthz declares no security requirement,
// so the auth middleware never reaches a resolver here.
type noopResolver struct{}

func (noopResolver) Resolve(context.Context, string) (auth.Identity, error) {
	return auth.Identity{}, auth.ErrUnauthenticated
}

func TestServer_RequestID_EchoedFromRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	e, humaAPI := NewServer(logger, noopResolver{}, ProvideAuthConfig())
	RegisterHealthz(humaAPI, logger)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(echo.HeaderXRequestID, "client-supplied-id")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if got := rec.Header().Get(echo.HeaderXRequestID); got != "client-supplied-id" {
		t.Fatalf("expected client-supplied request id to be echoed back, got %q", got)
	}
}

func TestServer_RequestID_GeneratedWhenAbsent(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	e, humaAPI := NewServer(logger, noopResolver{}, ProvideAuthConfig())
	RegisterHealthz(humaAPI, logger)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if got := rec.Header().Get(echo.HeaderXRequestID); got == "" {
		t.Fatal("expected a generated request id, got an empty header")
	}
}

func TestServer_LoggingMiddleware_LogsRequestLine(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	e, humaAPI := NewServer(logger, noopResolver{}, ProvideAuthConfig())
	RegisterHealthz(humaAPI, logger)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry); err != nil {
		t.Fatalf("expected exactly one JSON log line, got %q: %v", buf.String(), err)
	}
	for _, key := range []string{"request_id", "status", "duration"} {
		if _, ok := entry[key]; !ok {
			t.Fatalf("expected log entry to contain %q, got %+v", key, entry)
		}
	}
	if entry["request_id"] != rec.Header().Get(echo.HeaderXRequestID) {
		t.Fatalf("expected logged request_id %v to match response header %q", entry["request_id"], rec.Header().Get(echo.HeaderXRequestID))
	}
}
