//go:build wireinject

package main

import (
	"context"

	"github.com/google/wire"

	"[PROJECT-NAME]/backend/example"
	"[PROJECT-NAME]/backend/internal/config"
	"[PROJECT-NAME]/backend/internal/database"
	"[PROJECT-NAME]/backend/internal/logging"
	"[PROJECT-NAME]/backend/internal/presentation/api"
	"[PROJECT-NAME]/backend/users"
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
