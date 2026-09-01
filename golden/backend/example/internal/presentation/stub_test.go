package presentation

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dennys-bd/gonext/auth"

	"golden-app/backend/example/internal/application"
	"golden-app/backend/example/internal/infrastructure/memory"
	apipkg "golden-app/backend/internal/presentation/api"
)

// rejectingResolver stands in for a provider with no valid sessions:
// the example track's tests care that the guard is wired, not that
// the users track works.
type rejectingResolver struct{}

func (rejectingResolver) Resolve(context.Context, string) (auth.Identity, error) {
	return auth.Identity{}, auth.ErrUnauthenticated
}

func newTestAPI(t *testing.T) (humatest.TestAPI, *application.StubService) {
	t.Helper()

	_, api := humatest.New(t)
	api.UseMiddleware(apipkg.NewAuthMiddleware(
		api,
		rejectingResolver{},
		apipkg.ProvideAuthConfig(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	))

	svc := application.NewStubService(memory.NewStubRepository())
	RegisterStub(api, svc)
	return api, svc
}

// Creating a stub is a write, so it requires a session; reading one
// does not. This is the reference every generated project copies.
func TestCreateStub_RequiresASession(t *testing.T) {
	api, _ := newTestAPI(t)

	resp := api.Post("/stubs", map[string]string{"name": "demo"})
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a session, got %d: %s", resp.Code, resp.Body.String())
	}
}

// GET /stubs/{id} declares no security requirement, so it stays
// reachable without a credential. The stub is seeded through the
// service rather than the API, since POST /stubs is now guarded.
func TestGetStub_IsPublic(t *testing.T) {
	api, svc := newTestAPI(t)

	stub, err := svc.CreateStub(context.Background(), "demo")
	if err != nil {
		t.Fatalf("seeding a stub: %v", err)
	}

	resp := api.Get("/stubs/" + stub.ID)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestGetStub_NotFound(t *testing.T) {
	api, _ := newTestAPI(t)

	resp := api.Get("/stubs/does-not-exist")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body.String())
	}
}
