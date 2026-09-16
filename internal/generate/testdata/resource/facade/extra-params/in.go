// Package users is the public facade for the users domain. Register
// is the only symbol other packages may call — everything else lives
// under users/internal and is compiler-sealed.
package users

import (
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dennys-bd/gonext/auth"
	"github.com/uptrace/bun"

	"golden-app/backend/internal/config"
	"golden-app/backend/internal/database"
	"golden-app/backend/users/domain"
	"golden-app/backend/users/internal/application"
	authadapter "golden-app/backend/users/internal/infrastructure/auth"
	"golden-app/backend/users/internal/infrastructure/notify"
	"golden-app/backend/users/internal/infrastructure/postgres"
	"golden-app/backend/users/internal/presentation"
)

// Register wires up the users domain's dependencies and registers its
// HTTP endpoints on api, backed by db. cfg supplies the environment
// that gates the domain's production-only behaviour, logger backs the
// default no-op notifier, and issuer is passed in rather than built
// here so the process has exactly one session issuer — the auth
// middleware must resolve against the same one that issues.
func Register(api huma.API, db *bun.DB, cfg config.Config, logger *slog.Logger, issuer domain.SessionIssuer) error {
	store := postgres.NewStore(db)
	tx := postgres.NewTxRunner(database.NewBunTransactor(db))
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

// ProvideSessionIssuer builds the Postgres-backed session issuer.
func ProvideSessionIssuer(db *bun.DB) domain.SessionIssuer {
	return postgres.NewSessionIssuer(db)
}

// ProvideResolver adapts the session issuer to gonext's published
// auth.Resolver contract.
//
// This provider is the Auth Provider Abstraction: pointing wire at a
// different auth.Resolver — a Clerk or Supabase adapter the project
// simply `go get`s — swaps the identity provider without touching the
// middleware, the contract, or any handler that reads an identity.
func ProvideResolver(issuer domain.SessionIssuer) auth.Resolver {
	return authadapter.NewResolver(issuer)
}

// ProvideRegistration calls Register and returns a marker wire can
// depend on to guarantee the registration ran.
func ProvideRegistration(api huma.API, db *bun.DB, cfg config.Config, logger *slog.Logger, issuer domain.SessionIssuer) (Registered, error) {
	if err := Register(api, db, cfg, logger, issuer); err != nil {
		return Registered{}, err
	}
	return Registered{}, nil
}
