package presentation

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dennys-bd/gonext/auth"

	"[PROJECT-NAME]/backend/example/domain"
	"[PROJECT-NAME]/backend/example/internal/application"
	"[PROJECT-NAME]/backend/example/internal/infrastructure/memory"
	apipkg "[PROJECT-NAME]/backend/internal/presentation/api"
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
	RegisterStub(api, svc, slog.New(slog.NewTextHandler(io.Discard, nil)))
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

// failingStubRepository always fails with driver-shaped text, standing
// in for a Postgres error the example track's group has no mapping
// for.
type failingStubRepository struct{}

func (failingStubRepository) Create(context.Context, domain.Stub) error {
	return nil
}

func (failingStubRepository) Get(context.Context, string) (domain.Stub, error) {
	return domain.Stub{}, errors.New(`pq: relation stubs does not exist`)
}

// The generic mapped/unmapped/ordering behaviour is proven once in
// httpx/errors_test.go; this is the per-track assertion that the
// example group is actually wired with a logger and does not leak —
// closing the defect this migration exists to fix.
func TestGetStub_UnmappedRepositoryErrorDoesNotLeak(t *testing.T) {
	const secretText = `pq: relation stubs does not exist`

	logs := &bytes.Buffer{}
	_, api := humatest.New(t)

	svc := application.NewStubService(failingStubRepository{})
	RegisterStub(api, svc, slog.New(slog.NewTextHandler(logs, nil)))

	resp := api.Get("/stubs/abc")
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", resp.Code, resp.Body.String())
	}
	if strings.Contains(resp.Body.String(), secretText) {
		t.Fatalf("expected the driver message not to reach the client, got %s", resp.Body.String())
	}
	if !strings.Contains(logs.String(), secretText) {
		t.Fatalf("expected the real error to be logged server-side, got %q", logs.String())
	}
}
