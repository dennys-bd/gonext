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

// productRejectingResolver stands in for a provider with no valid sessions.
type productRejectingResolver struct{}

func (productRejectingResolver) Resolve(context.Context, string) (auth.Identity, error) {
	return auth.Identity{}, auth.ErrUnauthenticated
}

// productAcceptingResolver treats any credential as a valid session.
type productAcceptingResolver struct{}

func (productAcceptingResolver) Resolve(context.Context, string) (auth.Identity, error) {
	return auth.Identity{UserID: "u-1"}, nil
}

// productSession is the header a secured request sends; the accepting
// resolver never looks at its value.
const productSession = "Cookie: " + auth.DefaultCookieName + "=x"

func newProductTestAPI(t *testing.T, resolver auth.Resolver) (humatest.TestAPI, *memory.ProductRepository) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	_, api := humatest.New(t)
	api.UseMiddleware(apipkg.NewAuthMiddleware(api, resolver, apipkg.ProvideAuthConfig(), logger))

	repo := memory.NewProductRepository()
	RegisterProduct(api, application.NewProductService(repo), logger)
	return api, repo
}

// seedProduct stores a product through the repository, not the API, so
// the read and write cases do not depend on the create endpoint.
func seedProduct(t *testing.T, repo *memory.ProductRepository) domain.Product {
	t.Helper()
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	product := domain.Product{
		ID:        "product-1",
		Title:     "demo",
		Price:     42,
		CreatedAt: at,
		UpdatedAt: at,
	}
	if err := repo.Create(context.Background(), product); err != nil {
		t.Fatalf("seeding a product: %v", err)
	}
	return product
}

func TestCreateProduct_RequiresASession(t *testing.T) {
	api, _ := newProductTestAPI(t, productRejectingResolver{})

	resp := api.Post("/products", map[string]any{"title": "demo", "price": 42})
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a session, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestCreateProduct(t *testing.T) {
	api, _ := newProductTestAPI(t, productAcceptingResolver{})

	resp := api.Post("/products", productSession, map[string]any{"title": "demo", "price": 42})
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestCreateProduct_MissingField(t *testing.T) {
	api, _ := newProductTestAPI(t, productAcceptingResolver{})

	resp := api.Post("/products", productSession, map[string]any{})
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestGetProduct_RequiresASession(t *testing.T) {
	api, _ := newProductTestAPI(t, productRejectingResolver{})

	resp := api.Get("/products/does-not-exist")
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a session, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestGetProduct(t *testing.T) {
	api, repo := newProductTestAPI(t, productAcceptingResolver{})
	product := seedProduct(t, repo)

	resp := api.Get("/products/"+product.ID, productSession)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestGetProduct_NotFound(t *testing.T) {
	api, _ := newProductTestAPI(t, productAcceptingResolver{})

	resp := api.Get("/products/does-not-exist", productSession)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestListProducts_RequiresASession(t *testing.T) {
	api, _ := newProductTestAPI(t, productRejectingResolver{})

	resp := api.Get("/products")
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a session, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestListProducts(t *testing.T) {
	api, repo := newProductTestAPI(t, productAcceptingResolver{})
	seedProduct(t, repo)

	resp := api.Get("/products", productSession)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestUpdateProduct_RequiresASession(t *testing.T) {
	api, _ := newProductTestAPI(t, productRejectingResolver{})

	resp := api.Put("/products/does-not-exist", map[string]any{"title": "demo-updated", "price": 43})
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a session, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestUpdateProduct(t *testing.T) {
	api, repo := newProductTestAPI(t, productAcceptingResolver{})
	product := seedProduct(t, repo)

	resp := api.Put("/products/"+product.ID, productSession, map[string]any{"title": "demo-updated", "price": 43})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestUpdateProduct_NotFound(t *testing.T) {
	api, _ := newProductTestAPI(t, productAcceptingResolver{})

	resp := api.Put("/products/does-not-exist", productSession, map[string]any{"title": "demo-updated", "price": 43})
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestDeleteProduct_RequiresASession(t *testing.T) {
	api, _ := newProductTestAPI(t, productRejectingResolver{})

	resp := api.Delete("/products/does-not-exist")
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a session, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestDeleteProduct(t *testing.T) {
	api, repo := newProductTestAPI(t, productAcceptingResolver{})
	product := seedProduct(t, repo)

	resp := api.Delete("/products/"+product.ID, productSession)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestDeleteProduct_NotFound(t *testing.T) {
	api, _ := newProductTestAPI(t, productAcceptingResolver{})

	resp := api.Delete("/products/does-not-exist", productSession)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body.String())
	}
}
