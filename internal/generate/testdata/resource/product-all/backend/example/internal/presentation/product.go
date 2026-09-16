package presentation

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dennys-bd/gonext/auth"

	"golden-app/backend/example/domain"
	"golden-app/backend/example/internal/application"
	"golden-app/backend/internal/presentation/httpx"
)

// productBody is the request body shared by create and update.
type productBody struct {
	Title string `json:"title" minLength:"1"`
	Price int    `json:"price"`
}

type productItem struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Price     int       `json:"price"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type createProductInput struct {
	Body productBody
}

type productIDInput struct {
	ID string `path:"id" doc:"Product id"`
}

type listProductsInput struct {
	Limit  int `query:"limit" default:"50" minimum:"1" maximum:"200"`
	Offset int `query:"offset" minimum:"0"`
}

type updateProductInput struct {
	ID   string `path:"id" doc:"Product id"`
	Body productBody
}

type productOutput struct {
	Body productItem
}

type listProductsOutput struct {
	Body struct {
		Items []productItem `json:"items"`
	}
}

type productHandlers struct {
	svc *application.ProductService
}

// RegisterProduct registers the product endpoints under /products on api,
// backed by svc. Unmapped errors are logged through logger and returned
// to the client as a flat 500.
func RegisterProduct(api huma.API, svc *application.ProductService, logger *slog.Logger) {
	h := &productHandlers{svc: svc}
	g := httpx.NewGroup(api, "/products", "Example", logger).Errors(
		httpx.Map(domain.ErrProductNotFound, http.StatusNotFound),
	)

	httpx.Post(g, "", "create-product", h.createProduct,
		httpx.Summary("Create a product"),
		httpx.Status(http.StatusCreated),
		httpx.Secured(auth.Required()))

	httpx.Get(g, "/{id}", "get-product", h.getProduct,
		httpx.Summary("Get a product by id"),
		httpx.Secured(auth.Required()))

	httpx.Get(g, "", "list-products", h.listProducts,
		httpx.Summary("List products"),
		httpx.Secured(auth.Required()))

	httpx.Put(g, "/{id}", "update-product", h.updateProduct,
		httpx.Summary("Replace a product"),
		httpx.Secured(auth.Required()))

	httpx.Delete(g, "/{id}", "delete-product", h.deleteProduct,
		httpx.Summary("Delete a product"),
		httpx.Status(http.StatusNoContent),
		httpx.Secured(auth.Required()))
}

func (h *productHandlers) createProduct(ctx *httpx.Ctx, in *createProductInput) (*productOutput, error) {
	product, err := h.svc.CreateProduct(ctx, in.Body.Title, in.Body.Price)
	if err != nil {
		return nil, err
	}
	return &productOutput{Body: toProductItem(product)}, nil
}

func (h *productHandlers) getProduct(ctx *httpx.Ctx, in *productIDInput) (*productOutput, error) {
	product, err := h.svc.GetProduct(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	return &productOutput{Body: toProductItem(product)}, nil
}

func (h *productHandlers) listProducts(ctx *httpx.Ctx, in *listProductsInput) (*listProductsOutput, error) {
	products, err := h.svc.ListProducts(ctx, in.Limit, in.Offset)
	if err != nil {
		return nil, err
	}
	out := &listProductsOutput{}
	out.Body.Items = make([]productItem, 0, len(products))
	for _, product := range products {
		out.Body.Items = append(out.Body.Items, toProductItem(product))
	}
	return out, nil
}

func (h *productHandlers) updateProduct(ctx *httpx.Ctx, in *updateProductInput) (*productOutput, error) {
	product, err := h.svc.UpdateProduct(ctx, in.ID, in.Body.Title, in.Body.Price)
	if err != nil {
		return nil, err
	}
	return &productOutput{Body: toProductItem(product)}, nil
}

func (h *productHandlers) deleteProduct(ctx *httpx.Ctx, in *productIDInput) (*struct{}, error) {
	if err := h.svc.DeleteProduct(ctx, in.ID); err != nil {
		return nil, err
	}
	return &struct{}{}, nil
}

func toProductItem(product domain.Product) productItem {
	return productItem{
		ID:        product.ID,
		Title:     product.Title,
		Price:     product.Price,
		CreatedAt: product.CreatedAt,
		UpdatedAt: product.UpdatedAt,
	}
}
