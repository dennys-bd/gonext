// Package example is the public facade for the example domain.
package example

import (
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/uptrace/bun"
)

// Register wires up the example domain.
func Register(a huma.API, conn *bun.DB, log *slog.Logger) error {
	return nil
}
