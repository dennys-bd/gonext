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

// widgetBody is the request body shared by create and update.
type widgetBody struct {
	Name   string    `json:"name" minLength:"1"`
	Count  int       `json:"count"`
	Total  int64     `json:"total" format:"int64"`
	Ratio  float64   `json:"ratio"`
	Active bool      `json:"active"`
	Due    time.Time `json:"due" format:"date-time"`
}

type widgetItem struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Count     int       `json:"count"`
	Total     int64     `json:"total"`
	Ratio     float64   `json:"ratio"`
	Active    bool      `json:"active"`
	Due       time.Time `json:"due"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type createWidgetInput struct {
	Body widgetBody
}

type widgetIDInput struct {
	ID string `path:"id" doc:"Widget id"`
}

type widgetOutput struct {
	Body widgetItem
}

type widgetHandlers struct {
	svc *application.WidgetService
}

// RegisterWidget registers the widget endpoints under /widgets on api,
// backed by svc. Unmapped errors are logged through logger and returned
// to the client as a flat 500.
func RegisterWidget(api huma.API, svc *application.WidgetService, logger *slog.Logger) {
	h := &widgetHandlers{svc: svc}
	g := httpx.NewGroup(api, "/widgets", "Example", logger).Errors(
		httpx.Map(domain.ErrWidgetNotFound, http.StatusNotFound),
	)

	httpx.Post(g, "", "create-widget", h.createWidget,
		httpx.Summary("Create a widget"),
		httpx.Status(http.StatusCreated))

	httpx.Get(g, "/{id}", "get-widget", h.getWidget,
		httpx.Summary("Get a widget by id"))
}

func (h *widgetHandlers) createWidget(ctx *httpx.Ctx, in *createWidgetInput) (*widgetOutput, error) {
	widget, err := h.svc.CreateWidget(ctx, in.Body.Name, in.Body.Count, in.Body.Total, in.Body.Ratio, in.Body.Active, in.Body.Due)
	if err != nil {
		return nil, err
	}
	return &widgetOutput{Body: toWidgetItem(widget)}, nil
}

func (h *widgetHandlers) getWidget(ctx *httpx.Ctx, in *widgetIDInput) (*widgetOutput, error) {
	widget, err := h.svc.GetWidget(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	return &widgetOutput{Body: toWidgetItem(widget)}, nil
}

func toWidgetItem(widget domain.Widget) widgetItem {
	return widgetItem{
		ID:        widget.ID,
		Name:      widget.Name,
		Count:     widget.Count,
		Total:     widget.Total,
		Ratio:     widget.Ratio,
		Active:    widget.Active,
		Due:       widget.Due,
		CreatedAt: widget.CreatedAt,
		UpdatedAt: widget.UpdatedAt,
	}
}
