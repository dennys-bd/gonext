package presentation

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dennys-bd/gonext/auth"

	"golden-app/backend/example/domain"
	"golden-app/backend/example/internal/application"
	"golden-app/backend/example/internal/infrastructure/memory"
	apipkg "golden-app/backend/internal/presentation/api"
)

// tagRejectingResolver stands in for a provider with no valid sessions.
type tagRejectingResolver struct{}

func (tagRejectingResolver) Resolve(context.Context, string) (auth.Identity, error) {
	return auth.Identity{}, auth.ErrUnauthenticated
}

// tagAcceptingResolver treats any credential as a valid session.
type tagAcceptingResolver struct{}

func (tagAcceptingResolver) Resolve(context.Context, string) (auth.Identity, error) {
	return auth.Identity{UserID: "u-1"}, nil
}

// tagSession is the header a secured request sends; the accepting
// resolver never looks at its value.
const tagSession = "Cookie: " + auth.DefaultCookieName + "=x"

func newTagTestAPI(t *testing.T, resolver auth.Resolver) (humatest.TestAPI, *memory.TagRepository) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	_, api := humatest.New(t)
	api.UseMiddleware(apipkg.NewAuthMiddleware(api, resolver, apipkg.ProvideAuthConfig(), logger))

	repo := memory.NewTagRepository()
	RegisterTag(api, application.NewTagService(repo), logger)
	return api, repo
}

// seedTag stores a tag through the repository, not the API, so
// the read and write cases do not depend on the create endpoint.
func seedTag(t *testing.T, repo *memory.TagRepository) domain.Tag {
	t.Helper()
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tag := domain.Tag{
		ID:        "tag-1",
		CreatedAt: at,
		UpdatedAt: at,
	}
	if err := repo.Create(context.Background(), tag); err != nil {
		t.Fatalf("seeding a tag: %v", err)
	}
	return tag
}

func TestCreateTag_RequiresASession(t *testing.T) {
	api, _ := newTagTestAPI(t, tagRejectingResolver{})

	resp := api.Post("/tags", map[string]any{})
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a session, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestCreateTag(t *testing.T) {
	api, _ := newTagTestAPI(t, tagAcceptingResolver{})

	resp := api.Post("/tags", tagSession, map[string]any{})
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestGetTag_RequiresASession(t *testing.T) {
	api, _ := newTagTestAPI(t, tagRejectingResolver{})

	resp := api.Get("/tags/does-not-exist")
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a session, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestGetTag(t *testing.T) {
	api, repo := newTagTestAPI(t, tagAcceptingResolver{})
	tag := seedTag(t, repo)

	resp := api.Get("/tags/"+tag.ID, tagSession)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestGetTag_NotFound(t *testing.T) {
	api, _ := newTagTestAPI(t, tagAcceptingResolver{})

	resp := api.Get("/tags/does-not-exist", tagSession)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestListTags_RequiresASession(t *testing.T) {
	api, _ := newTagTestAPI(t, tagRejectingResolver{})

	resp := api.Get("/tags")
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a session, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestListTags(t *testing.T) {
	api, repo := newTagTestAPI(t, tagAcceptingResolver{})
	seedTag(t, repo)

	resp := api.Get("/tags", tagSession)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestUpdateTag_RequiresASession(t *testing.T) {
	api, _ := newTagTestAPI(t, tagRejectingResolver{})

	resp := api.Put("/tags/does-not-exist", map[string]any{})
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a session, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestUpdateTag(t *testing.T) {
	api, repo := newTagTestAPI(t, tagAcceptingResolver{})
	tag := seedTag(t, repo)

	resp := api.Put("/tags/"+tag.ID, tagSession, map[string]any{})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestUpdateTag_NotFound(t *testing.T) {
	api, _ := newTagTestAPI(t, tagAcceptingResolver{})

	resp := api.Put("/tags/does-not-exist", tagSession, map[string]any{})
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestDeleteTag_RequiresASession(t *testing.T) {
	api, _ := newTagTestAPI(t, tagRejectingResolver{})

	resp := api.Delete("/tags/does-not-exist")
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a session, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestDeleteTag(t *testing.T) {
	api, repo := newTagTestAPI(t, tagAcceptingResolver{})
	tag := seedTag(t, repo)

	resp := api.Delete("/tags/"+tag.ID, tagSession)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestDeleteTag_NotFound(t *testing.T) {
	api, _ := newTagTestAPI(t, tagAcceptingResolver{})

	resp := api.Delete("/tags/does-not-exist", tagSession)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body.String())
	}
}
