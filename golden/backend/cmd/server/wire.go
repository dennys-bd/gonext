//go:build wireinject

package main

import (
	"context"

	"github.com/google/wire"

	"golden-app/backend/internal/config"
	"golden-app/backend/internal/database"
	"golden-app/backend/internal/logging"
	"golden-app/backend/internal/presentation/api"
	"golden-app/backend/tracks"
	"golden-app/backend/users"
)

// InitializeApp builds the full dependency graph — config, logger, DB
// pool, HTTP server, and every domain's endpoint registration — and
// returns the assembled App, a cleanup func (closing the DB pool),
// and any error hit along the way. Run `wire ./backend/cmd/server/...`
// (see `make generate`) to regenerate wire_gen.go after changing the
// provider list below.
func InitializeApp(ctx context.Context) (*App, func(), error) {
	wire.Build(
		config.Load,
		logging.New,
		wire.FieldsOf(new(config.Config), "DatabaseURL"),
		database.ProvideDB,
		api.NewEcho,
		api.NewHumaAPI,
		api.ProvideHealthzRegistration,
		api.ProvideReadyzRegistration,
		tracks.ProvideRegistration,
		users.ProvideRegistration,
		NewApp,
	)
	return nil, nil, nil
}
