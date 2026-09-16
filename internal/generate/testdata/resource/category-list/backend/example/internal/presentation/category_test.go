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

// categoryRejectingResolver stands in for a provider with no valid sessions.
type categoryRejectingResolver struct{}

func (categoryRejectingResolver) Resolve(context.Context, string) (auth.Identity, error) {
	return auth.Identity{}, auth.ErrUnauthenticated
}

func newCategoryTestAPI(t *testing.T, resolver auth.Resolver) (humatest.TestAPI, *memory.CategoryRepository) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	_, api := humatest.New(t)
	api.UseMiddleware(apipkg.NewAuthMiddleware(api, resolver, apipkg.ProvideAuthConfig(), logger))

	repo := memory.NewCategoryRepository()
	RegisterCategory(api, application.NewCategoryService(repo), logger)
	return api, repo
}

// seedCategory stores a category through the repository, not the API, so
// the read and write cases do not depend on the create endpoint.
func seedCategory(t *testing.T, repo *memory.CategoryRepository) domain.Category {
	t.Helper()
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	category := domain.Category{
		ID:        "category-1",
		Name:      "demo",
		CreatedAt: at,
		UpdatedAt: at,
	}
	if err := repo.Create(context.Background(), category); err != nil {
		t.Fatalf("seeding a category: %v", err)
	}
	return category
}

func TestListCategories(t *testing.T) {
	api, repo := newCategoryTestAPI(t, categoryRejectingResolver{})
	seedCategory(t, repo)

	resp := api.Get("/categories")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}
