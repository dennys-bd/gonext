// Package migrate applies a generated project's pending Postgres
// migrations from this repo, without vendoring a migrate command into
// every scaffolded project.
package migrate

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	xexec "github.com/dennys-bd/gonext/internal/exec"
	"github.com/dennys-bd/gonext/internal/project"
)

// runnerFilename is the fixed, gitignored name the temp runner is
// materialized under in the target project's backend/.
const runnerFilename = "gonext_migrate_runner.go"

// MigrationImportsToken is the placeholder the runner template
// holds where one blank import per backend/<domain>/migrations
// directory goes.
const MigrationImportsToken = "[MIGRATION-IMPORTS]"

// domainNameRE is the domain character set dbmigrate.Register accepts.
var domainNameRE = regexp.MustCompile(`^[a-z0-9_]+$`)

//go:embed runner_template.go.tmpl
var runnerTemplate []byte

// runFunc executes the materialized runner; overridden in tests to
// avoid actually invoking `go run` against a real database.
var runFunc = xexec.Run

// Apply materializes the migration runner under root/backend with
// root's module path and its migration imports substituted in, runs
// it via `go run`, and removes it afterward regardless of outcome.
func Apply(ctx context.Context, root string) error {
	imports, err := migrationImports(root)
	if err != nil {
		return err
	}

	extra := map[string]string{MigrationImportsToken: imports}
	backendDir, remove, err := project.MaterializeRunner(root, runnerFilename, runnerTemplate, extra)
	if err != nil {
		return err
	}
	defer remove()

	if err := runFunc(ctx, backendDir, "go", "run", runnerFilename); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}
	return nil
}

// migrationImports scans root/backend/*/migrations/ and renders one
// blank-import line per directory holding at least one non-test
// Go file, sorted by domain. backend/internal is never a domain.
// No such directory at all renders "" — the runner then imports
// nothing and Apply reports nothing to run.
func migrationImports(root string) (string, error) {
	backendDir := filepath.Join(root, "backend")
	entries, err := os.ReadDir(backendDir)
	if err != nil {
		return "", fmt.Errorf("scanning backend domains: %w", err)
	}

	var b strings.Builder
	for _, entry := range entries {
		// Only a name dbmigrate's path regex accepts can be a domain;
		// anything else could not register, and its name is rendered
		// verbatim into the runner's Go source.
		if !entry.IsDir() || entry.Name() == "internal" || !domainNameRE.MatchString(entry.Name()) {
			continue
		}
		hasMigration, err := hasMigrationFile(filepath.Join(backendDir, entry.Name(), "migrations"))
		if err != nil {
			return "", err
		}
		if !hasMigration {
			continue
		}
		b.WriteString("\t_ \"" + project.ModulePathToken + "/backend/" + entry.Name() + "/migrations\"\n")
	}
	return b.String(), nil
}

// hasMigrationFile reports whether dir holds at least one non-test Go
// file. A missing dir is not an error: the domain simply has no
// migrations yet.
func hasMigrationFile(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("scanning %s: %w", dir, err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			return true, nil
		}
	}
	return false, nil
}
