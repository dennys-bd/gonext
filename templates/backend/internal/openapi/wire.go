//go:build wireinject

package openapi

import (
	"log/slog"

	"github.com/google/wire"
	"github.com/uptrace/bun"

	"[PROJECT-NAME]/backend/example"
	"[PROJECT-NAME]/backend/internal/config"
	"[PROJECT-NAME]/backend/internal/presentation/api"
	"[PROJECT-NAME]/backend/users"
)

// Initialize builds the same Huma API InitializeApp builds — every
// domain registered through the same Register calls — over the config,
// logger and database the caller supplies. `gonext openapi` passes a
// fixed config, a discarding logger and a *bun.DB that never connects;
// registration only constructs repositories that hold the handle, so
// nothing dials. Run `gonext generate` after changing the provider list
// below.
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
