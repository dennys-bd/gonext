package api

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
)

type fakePinger struct {
	err error
}

func (f fakePinger) PingContext(context.Context) error {
	return f.err
}

func TestReadyz_OK(t *testing.T) {
	_, api := humatest.New(t)
	RegisterReadyz(api, fakePinger{}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	resp := api.Get("/readyz")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", resp.Body.String())
	}
}

// A ping failure reports through readyzOutput's own Status field with
// a nil error, not through the error return — the group wrapper only
// post-processes a non-nil error, so this 503 must survive the move
// onto httpx.Get unchanged.
func TestReadyz_Unavailable(t *testing.T) {
	_, api := humatest.New(t)
	RegisterReadyz(api, fakePinger{err: errors.New("connection refused")}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	resp := api.Get("/readyz")
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), `"status":"unavailable"`) {
		t.Fatalf("unexpected body: %s", resp.Body.String())
	}
}
