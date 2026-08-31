package migrate

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	xexec "github.com/dennys-bd/gonext/internal/exec"
)

const modulePathToken = "[MODULE-PATH]"

// runnerFilename is the fixed, gitignored name the temp runner is
// materialized under at the target project's root. It must not start
// with "." or "_": the go tool silently excludes such files, which
// makes `go run` report "no Go files" instead of running it.
const runnerFilename = "gonext_migrate_runner.go"

const runnerFileMode = 0o644

//go:embed runner_template.go.tmpl
var runnerTemplate []byte

// runFunc executes the materialized runner; overridden in tests to
// avoid actually invoking `go run` against a real database.
var runFunc = xexec.Run

// Apply resolves root's module path, materializes a temporary runner
// file under root/backend (so it can see backend's internal/
// packages — Go's internal-import rule would block a file at root
// itself) with the module path substituted in, runs it via `go run`,
// and removes it afterward regardless of outcome.
func Apply(ctx context.Context, root string) error {
	modulePath, err := ModulePath(root)
	if err != nil {
		return fmt.Errorf("resolving module path: %w", err)
	}

	backendDir := filepath.Join(root, "backend")
	runnerPath := filepath.Join(backendDir, runnerFilename)
	content := bytes.ReplaceAll(runnerTemplate, []byte(modulePathToken), []byte(modulePath))
	if err := os.WriteFile(runnerPath, content, runnerFileMode); err != nil {
		return fmt.Errorf("writing migration runner: %w", err)
	}
	defer os.Remove(runnerPath)

	if err := runFunc(ctx, backendDir, "go", "run", runnerFilename); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}
	return nil
}
