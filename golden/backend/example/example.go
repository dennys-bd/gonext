// Package example is the public facade for the example domain. Register
// is the only symbol other packages may call — everything else lives
// under example/internal and is compiler-sealed.
package example

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/uptrace/bun"

	"golden-app/backend/example/internal/application"
	"golden-app/backend/example/internal/infrastructure/postgres"
	"golden-app/backend/example/internal/presentation"
)

// Register wires up the example domain's dependencies and registers its
// HTTP endpoints on api, backed by db.
func Register(api huma.API, db *bun.DB) error {
	repo := postgres.NewStubRepository(db)
	svc := application.NewStubService(repo)
	presentation.RegisterStub(api, svc)
	return nil
}

// Registered is a marker type with no fields: its only purpose is to
// give wire something to depend on, so it sequences Register's
// side-effecting call into the generated injector instead of main.go
// calling it by hand.
type Registered struct{}

// ProvideRegistration calls Register and returns a marker wire can
// depend on to guarantee the registration ran.
func ProvideRegistration(api huma.API, db *bun.DB) (Registered, error) {
	if err := Register(api, db); err != nil {
		return Registered{}, err
	}
	return Registered{}, nil
}
