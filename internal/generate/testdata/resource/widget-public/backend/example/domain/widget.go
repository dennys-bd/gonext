package domain

import (
	"context"
	"errors"
	"time"
)

// ErrWidgetNotFound is returned when a Widget cannot be found by id.
var ErrWidgetNotFound = errors.New("example: widget not found")

// Widget is the example domain's widget entity.
type Widget struct {
	ID        string
	Name      string
	Count     int
	Total     int64
	Ratio     float64
	Active    bool
	Due       time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// WidgetRepository persists and retrieves Widgets.
type WidgetRepository interface {
	Create(ctx context.Context, widget Widget) error
	Get(ctx context.Context, id string) (Widget, error)
}
