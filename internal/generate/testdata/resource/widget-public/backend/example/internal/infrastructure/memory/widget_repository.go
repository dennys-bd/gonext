package memory

import (
	"context"
	"sync"

	"golden-app/backend/example/domain"
)

var _ domain.WidgetRepository = (*WidgetRepository)(nil)

// WidgetRepository is an in-memory, mutex-guarded domain.WidgetRepository.
type WidgetRepository struct {
	mu      sync.Mutex
	widgets map[string]domain.Widget
}

// NewWidgetRepository constructs an empty in-memory WidgetRepository.
func NewWidgetRepository() *WidgetRepository {
	return &WidgetRepository{widgets: make(map[string]domain.Widget)}
}

// Create stores widget, keyed by its ID.
func (r *WidgetRepository) Create(_ context.Context, widget domain.Widget) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.widgets[widget.ID] = widget
	return nil
}

// Get retrieves the Widget with the given id, or domain.ErrWidgetNotFound.
func (r *WidgetRepository) Get(_ context.Context, id string) (domain.Widget, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	widget, ok := r.widgets[id]
	if !ok {
		return domain.Widget{}, domain.ErrWidgetNotFound
	}
	return widget, nil
}
