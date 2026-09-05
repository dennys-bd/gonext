package api

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
)

func TestHealthz(t *testing.T) {
	_, api := humatest.New(t)
	RegisterHealthz(api, slog.New(slog.NewTextHandler(io.Discard, nil)))

	resp := api.Get("/healthz")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", resp.Body.String())
	}
}
