package presentation

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"golden-app/backend/example/domain"
	"golden-app/backend/example/internal/application"
	"golden-app/backend/internal/presentation/httpx"
)

type categoryItem struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type listCategoriesInput struct {
	Limit  int `query:"limit" default:"50" minimum:"1" maximum:"200"`
	Offset int `query:"offset" minimum:"0"`
}

type listCategoriesOutput struct {
	Body struct {
		Items []categoryItem `json:"items"`
	}
}

type categoryHandlers struct {
	svc *application.CategoryService
}

// RegisterCategory registers the category endpoints under /categories on api,
// backed by svc. Unmapped errors are logged through logger and returned
// to the client as a flat 500.
func RegisterCategory(api huma.API, svc *application.CategoryService, logger *slog.Logger) {
	h := &categoryHandlers{svc: svc}
	g := httpx.NewGroup(api, "/categories", "Example", logger).Errors(
		httpx.Map(domain.ErrCategoryNotFound, http.StatusNotFound),
	)

	httpx.Get(g, "", "list-categories", h.listCategories,
		httpx.Summary("List categories"))
}

func (h *categoryHandlers) listCategories(ctx *httpx.Ctx, in *listCategoriesInput) (*listCategoriesOutput, error) {
	categories, err := h.svc.ListCategories(ctx, in.Limit, in.Offset)
	if err != nil {
		return nil, err
	}
	out := &listCategoriesOutput{}
	out.Body.Items = make([]categoryItem, 0, len(categories))
	for _, category := range categories {
		out.Body.Items = append(out.Body.Items, toCategoryItem(category))
	}
	return out, nil
}

func toCategoryItem(category domain.Category) categoryItem {
	return categoryItem{
		ID:        category.ID,
		Name:      category.Name,
		CreatedAt: category.CreatedAt,
		UpdatedAt: category.UpdatedAt,
	}
}
