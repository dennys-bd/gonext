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

// widgetRejectingResolver stands in for a provider with no valid sessions.
type widgetRejectingResolver struct{}

func (widgetRejectingResolver) Resolve(context.Context, string) (auth.Identity, error) {
	return auth.Identity{}, auth.ErrUnauthenticated
}

func newWidgetTestAPI(t *testing.T, resolver auth.Resolver) (humatest.TestAPI, *memory.WidgetRepository) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	_, api := humatest.New(t)
	api.UseMiddleware(apipkg.NewAuthMiddleware(api, resolver, apipkg.ProvideAuthConfig(), logger))

	repo := memory.NewWidgetRepository()
	RegisterWidget(api, application.NewWidgetService(repo), logger)
	return api, repo
}

// seedWidget stores a widget through the repository, not the API, so
// the read and write cases do not depend on the create endpoint.
func seedWidget(t *testing.T, repo *memory.WidgetRepository) domain.Widget {
	t.Helper()
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	widget := domain.Widget{
		ID:        "widget-1",
		Name:      "demo",
		Count:     42,
		Total:     int64(42),
		Ratio:     9.5,
		Active:    true,
		Due:       time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		CreatedAt: at,
		UpdatedAt: at,
	}
	if err := repo.Create(context.Background(), widget); err != nil {
		t.Fatalf("seeding a widget: %v", err)
	}
	return widget
}

func TestCreateWidget(t *testing.T) {
	api, _ := newWidgetTestAPI(t, widgetRejectingResolver{})

	resp := api.Post("/widgets", map[string]any{"name": "demo", "count": 42, "total": 42, "ratio": 9.5, "active": true, "due": "2026-01-02T03:04:05Z"})
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestCreateWidget_MissingField(t *testing.T) {
	api, _ := newWidgetTestAPI(t, widgetRejectingResolver{})

	resp := api.Post("/widgets", map[string]any{})
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestGetWidget(t *testing.T) {
	api, repo := newWidgetTestAPI(t, widgetRejectingResolver{})
	widget := seedWidget(t, repo)

	resp := api.Get("/widgets/" + widget.ID)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestGetWidget_NotFound(t *testing.T) {
	api, _ := newWidgetTestAPI(t, widgetRejectingResolver{})

	resp := api.Get("/widgets/does-not-exist")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body.String())
	}
}
