//go:build wireinject

package main

import (
	"context"

	"github.com/google/wire"

	"golden-app/backend/example"
	"golden-app/backend/internal/config"
	"golden-app/backend/internal/database"
	"golden-app/backend/internal/logging"
	"golden-app/backend/internal/presentation/api"
	"golden-app/backend/users"
)

// InitializeApp builds the full dependency graph — config, logger, DB
// pool, HTTP server, and every domain's endpoint registration — and
// returns the assembled App, a cleanup func (closing the DB pool),
// and any error hit along the way. Run `wire ./backend/...`
// (see `make generate`) to regenerate wire_gen.go after changing the
// provider list below.
func InitializeApp(ctx context.Context) (*App, func(), error) {
	wire.Build(
		config.Load,
		logging.New,
		wire.FieldsOf(new(config.Config), "DatabaseURL"),
		database.ProvideDB,
		api.ProvideAuthConfig,
		users.ProvideSessionIssuer,
		users.ProvideResolver,
		api.NewEcho,
		api.NewHumaAPI,
		api.ProvideHealthzRegistration,
		api.ProvideReadyzRegistration,
		example.ProvideRegistration,
		users.ProvideRegistration,
		NewApp,
	)
	return nil, nil, nil
}

// InitializeSpec builds the same Huma API InitializeApp builds —
// every domain registered through the same Register calls — over
// infrastructure that never connects, so the OpenAPI document can be
// produced with no database and no environment. Run `wire
// ./backend/...` (see `make generate`) to regenerate wire_gen.go
// after changing the provider list below.
func InitializeSpec() (*Spec, error) {
	wire.Build(
		specConfig,
		discardLogger,
		database.OfflineDB,
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
