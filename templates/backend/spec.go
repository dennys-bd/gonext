package main

import (
	"io"
	"log/slog"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"[PROJECT-NAME]/backend/example"
	"[PROJECT-NAME]/backend/internal/config"
	"[PROJECT-NAME]/backend/internal/presentation/api"
	"[PROJECT-NAME]/backend/users"
)

// Spec bundles what InitializeSpec builds: the Huma API with every
// domain registered, over infrastructure that never connects. It
// mirrors App's shape but carries only what producing the OpenAPI
// document needs.
type Spec struct {
	API huma.API
}

// NewSpec assembles Spec. Its marker parameters (HealthzRegistered,
// ReadyzRegistered, example.Registered, users.Registered) aren't
// read; depending on them is what forces wire to run every
// registration before InitializeSpec returns — the same trick
// NewApp uses.
func NewSpec(
	humaAPI huma.API,
	_ api.HealthzRegistered,
	_ api.ReadyzRegistered,
	_ example.Registered,
	_ users.Registered,
) *Spec {
	return &Spec{API: humaAPI}
}

// specConfig is a fixed Config for InitializeSpec, used instead of
// config.Load so the document depends only on the registrations, not
// on whatever environment variables happen to be set when it's
// produced. Env is deliberately "prod": users.Register derives
// cookie policy and relaxed-environment behavior from it, and the
// published contract should describe the production shape rather
// than whatever is convenient locally.
func specConfig() config.Config {
	return config.Config{
		Env:             "prod",
		Port:            8080,
		LogLevel:        "info",
		LogFormat:       "json",
		ShutdownTimeout: 10 * time.Second,
	}
}

// discardLogger is the logger InitializeSpec wires in. --openapi
// writes the document to stdout, and logging.New also writes to
// stdout — a single log line would corrupt the yaml, so registration
// logs here go nowhere instead.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
