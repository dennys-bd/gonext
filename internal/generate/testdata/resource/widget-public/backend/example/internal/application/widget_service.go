package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"golden-app/backend/example/domain"
)

// WidgetService implements the example domain's widget use cases.
type WidgetService struct {
	repo domain.WidgetRepository
}

// NewWidgetService constructs a WidgetService backed by repo.
func NewWidgetService(repo domain.WidgetRepository) *WidgetService {
	return &WidgetService{repo: repo}
}

// CreateWidget creates and persists a new Widget.
func (s *WidgetService) CreateWidget(ctx context.Context, name string, count int, total int64, ratio float64, active bool, due time.Time) (domain.Widget, error) {
	now := time.Now().UTC()
	widget := domain.Widget{
		ID:        newWidgetID(),
		Name:      name,
		Count:     count,
		Total:     total,
		Ratio:     ratio,
		Active:    active,
		Due:       due,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, widget); err != nil {
		return domain.Widget{}, err
	}
	return widget, nil
}

// GetWidget retrieves a Widget by id.
func (s *WidgetService) GetWidget(ctx context.Context, id string) (domain.Widget, error) {
	return s.repo.Get(ctx, id)
}

// newWidgetID generates a random hex id.
func newWidgetID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
