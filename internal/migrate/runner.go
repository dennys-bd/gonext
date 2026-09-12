// Package migrate applies a generated project's pending Postgres
// migrations from this repo, without vendoring a migrate command into
// every scaffolded project.
package migrate

import (
	"context"
	_ "embed"
	"fmt"

	xexec "github.com/dennys-bd/gonext/internal/exec"
	"github.com/dennys-bd/gonext/internal/project"
)

// runnerFilename is the fixed, gitignored name the temp runner is
// materialized under in the target project's backend/.
const runnerFilename = "gonext_migrate_runner.go"

//go:embed runner_template.go.tmpl
var runnerTemplate []byte

// runFunc executes the materialized runner; overridden in tests to
// avoid actually invoking `go run` against a real database.
var runFunc = xexec.Run

// Apply materializes the migration runner under root/backend with
// root's module path substituted in, runs it via `go run`, and
// removes it afterward regardless of outcome.
func Apply(ctx context.Context, root string) error {
	backendDir, remove, err := project.MaterializeRunner(root, runnerFilename, runnerTemplate)
	if err != nil {
		return err
	}
	defer remove()

	if err := runFunc(ctx, backendDir, "go", "run", runnerFilename); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}
	return nil
}
