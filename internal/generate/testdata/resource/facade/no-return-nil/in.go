package example

import (
	"fmt"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/uptrace/bun"
)

// Register wires up the example domain.
func Register(api huma.API, db *bun.DB, logger *slog.Logger) error {
	return fmt.Errorf("not yet")
}
