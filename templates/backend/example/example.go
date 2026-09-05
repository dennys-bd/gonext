// Package example is the public facade for the example domain. Register
// is the only symbol other packages may call — everything else lives
// under example/internal and is compiler-sealed.
package example

import (
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/uptrace/bun"

	"[PROJECT-NAME]/backend/example/internal/application"
	"[PROJECT-NAME]/backend/example/internal/infrastructure/postgres"
	"[PROJECT-NAME]/backend/example/internal/presentation"
)

// Register wires up the example domain's dependencies and registers
// its HTTP endpoints on api, backed by db. Unmapped errors are logged
// through logger.
func Register(api huma.API, db *bun.DB, logger *slog.Logger) error {
	repo := postgres.NewStubRepository(db)
	svc := application.NewStubService(repo)
	presentation.RegisterStub(api, svc, logger)
	return nil
}

// Registered is a marker type with no fields: its only purpose is to
// give wire something to depend on, so it sequences Register's
// side-effecting call into the generated injector instead of main.go
// calling it by hand.
type Registered struct{}

// ProvideRegistration calls Register and returns a marker wire can
// depend on to guarantee the registration ran.
func ProvideRegistration(api huma.API, db *bun.DB, logger *slog.Logger) (Registered, error) {
	if err := Register(api, db, logger); err != nil {
		return Registered{}, err
	}
	return Registered{}, nil
}
