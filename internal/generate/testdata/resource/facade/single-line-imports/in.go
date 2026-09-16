// Package example is the public facade for the example domain.
package example

import "log/slog"

import "github.com/danielgtaylor/huma/v2"
import "github.com/uptrace/bun"

// Register wires up the example domain.
func Register(api huma.API, db *bun.DB, logger *slog.Logger) error {
	return nil
}
