package example

import (
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/uptrace/bun"
)

// Mount wires up the example domain.
func Mount(api huma.API, db *bun.DB, logger *slog.Logger) error {
	return nil
}
