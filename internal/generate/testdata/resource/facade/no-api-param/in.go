package example

import (
	"log/slog"

	"github.com/uptrace/bun"
)

// Register wires up the example domain.
func Register(db *bun.DB, logger *slog.Logger) error {
	return nil
}
