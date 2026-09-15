//go:build wireinject

package openapi

import (
	"log/slog"

	"github.com/google/wire"
	"github.com/uptrace/bun"

	"golden-app/backend/example"
	"golden-app/backend/internal/config"
	"golden-app/backend/internal/presentation/api"
	"golden-app/backend/users"
)

// Initialize builds the same Huma API InitializeApp builds, over a fixed
// config and a *bun.DB that never dials — registration only constructs
// repositories. Run `gonext generate` after editing the provider list below.
func Initialize(cfg config.Config, logger *slog.Logger, db *bun.DB) (*Spec, error) {
	wire.Build(
		api.ProvideAuthConfig,
		users.ProvideSessionIssuer,
		users.ProvideResolver,
		api.NewEcho,
		api.NewHumaAPI,
		api.ProvideHealthzRegistration,
		api.ProvideReadyzRegistration,
		example.ProvideRegistration,
		users.ProvideRegistration,
		NewSpec,
	)
	return nil, nil
}
