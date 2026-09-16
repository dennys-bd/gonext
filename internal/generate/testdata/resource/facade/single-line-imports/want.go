// Package example is the public facade for the example domain.
package example

import "log/slog"

import "github.com/danielgtaylor/huma/v2"
import "github.com/uptrace/bun"

import (
	"golden-app/backend/example/internal/application"
	"golden-app/backend/example/internal/infrastructure/postgres"
	"golden-app/backend/example/internal/presentation"
)

// Register wires up the example domain.
func Register(api huma.API, db *bun.DB, logger *slog.Logger) error {
	productRepo := postgres.NewProductRepository(db)
	productSvc := application.NewProductService(productRepo)
	presentation.RegisterProduct(api, productSvc, logger)
	return nil
}
