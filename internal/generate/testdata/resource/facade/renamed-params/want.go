// Package example is the public facade for the example domain.
package example

import (
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/uptrace/bun"
	"golden-app/backend/example/internal/application"
	"golden-app/backend/example/internal/infrastructure/postgres"
	"golden-app/backend/example/internal/presentation"
)

// Register wires up the example domain.
func Register(a huma.API, conn *bun.DB, log *slog.Logger) error {
	productRepo := postgres.NewProductRepository(conn)
	productSvc := application.NewProductService(productRepo)
	presentation.RegisterProduct(a, productSvc, log)
	return nil
}
