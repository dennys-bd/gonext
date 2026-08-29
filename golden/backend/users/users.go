// Package users is the public facade for the users domain. Register
// is the only symbol other packages may call — everything else lives
// under users/internal and is compiler-sealed.
package users

import (
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/uptrace/bun"

	"golden-app/backend/internal/config"
	"golden-app/backend/internal/database"
	"golden-app/backend/users/internal/application"
	"golden-app/backend/users/internal/infrastructure/notify"
	"golden-app/backend/users/internal/infrastructure/postgres"
	"golden-app/backend/users/internal/presentation"
)

// Register wires up the users domain's dependencies and registers its
// HTTP endpoints on api, backed by db. cfg supplies the environment
// that gates the domain's production-only behaviour, and logger backs
// the default no-op notifier.
func Register(api huma.API, db *bun.DB, cfg config.Config, logger *slog.Logger) error {
	store := postgres.NewStore(db)
	tx := postgres.NewTxRunner(database.NewBunTransactor(db))
	issuer := postgres.NewSessionIssuer(db)
	notifier := notify.NewNoop(logger)

	svc := application.NewUserService(store, tx, issuer, notifier, application.NewArgon2Hasher(), cfg.Env)
	presentation.RegisterUsers(api, svc, presentation.NewCookieOptions(cfg.Env), logger)
	return nil
}

// Registered is a marker type with no fields: its only purpose is to
// give wire something to depend on, so it sequences Register's
// side-effecting call into the generated injector instead of main.go
// calling it by hand.
type Registered struct{}

// ProvideRegistration calls Register and returns a marker wire can
// depend on to guarantee the registration ran.
func ProvideRegistration(api huma.API, db *bun.DB, cfg config.Config, logger *slog.Logger) (Registered, error) {
	if err := Register(api, db, cfg, logger); err != nil {
		return Registered{}, err
	}
	return Registered{}, nil
}
