package api

import (
	"context"
	"errors"
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
	RegisterReadyz(api, fakePinger{})

	resp := api.Get("/readyz")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", resp.Body.String())
	}
}

func TestReadyz_Unavailable(t *testing.T) {
	_, api := humatest.New(t)
	RegisterReadyz(api, fakePinger{err: errors.New("connection refused")})

	resp := api.Get("/readyz")
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), `"status":"unavailable"`) {
		t.Fatalf("unexpected body: %s", resp.Body.String())
	}
}
