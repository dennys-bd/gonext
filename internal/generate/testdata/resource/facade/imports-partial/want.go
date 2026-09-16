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
func Register(api huma.API, db *bun.DB, logger *slog.Logger) error {
	svc := application.NewStubService(nil)
	presentation.RegisterStub(api, svc, logger)

	productRepo := postgres.NewProductRepository(db)
	productSvc := application.NewProductService(productRepo)
	presentation.RegisterProduct(api, productSvc, logger)
	return nil
}
