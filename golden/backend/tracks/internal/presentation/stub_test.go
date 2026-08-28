package presentation

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"

	"golden-app/backend/tracks/internal/application"
	"golden-app/backend/tracks/internal/infrastructure/memory"
)

func newTestAPI(t *testing.T) humatest.TestAPI {
	_, api := humatest.New(t)
	RegisterStub(api, application.NewStubService(memory.NewStubRepository()))
	return api
}

func TestCreateAndGetStub_RoundTrip(t *testing.T) {
	api := newTestAPI(t)

	createResp := api.Post("/stubs", map[string]string{"name": "demo"})
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", createResp.Code, createResp.Body.String())
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected non-empty id in response: %s", createResp.Body.String())
	}

	getResp := api.Get("/stubs/" + created.ID)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d: %s", getResp.Code, getResp.Body.String())
	}
}

func TestGetStub_NotFound(t *testing.T) {
	api := newTestAPI(t)
	resp := api.Get("/stubs/does-not-exist")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestCreateStub_EmptyName(t *testing.T) {
	api := newTestAPI(t)
	resp := api.Post("/stubs", map[string]string{"name": ""})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}
}
