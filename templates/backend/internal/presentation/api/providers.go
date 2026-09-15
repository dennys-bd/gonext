package api

import (
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/uptrace/bun"
)

// HealthzRegistered is a marker wire depends on to sequence RegisterHealthz's
// side-effecting call into the generated injector.
type HealthzRegistered struct{}

// ProvideHealthzRegistration registers /healthz and returns a marker
// wire can depend on to guarantee the registration ran.
func ProvideHealthzRegistration(api huma.API, logger *slog.Logger) HealthzRegistered {
	RegisterHealthz(api, logger)
	return HealthzRegistered{}
}

// ReadyzRegistered is a marker wire depends on to sequence RegisterReadyz's
// side-effecting call into the generated injector.
type ReadyzRegistered struct{}

// ProvideReadyzRegistration registers /readyz and returns a marker wire can
// depend on. It takes the concrete *bun.DB, not the unexported pinger
// interface, so wire.go in package main can wire it up.
func ProvideReadyzRegistration(api huma.API, db *bun.DB, logger *slog.Logger) ReadyzRegistered {
	RegisterReadyz(api, db, logger)
	return ReadyzRegistered{}
}
