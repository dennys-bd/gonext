// Package openapi produces a generated project's OpenAPI document
// from this repo, without vendoring the spec-dumping command into
// every scaffolded project: it materializes a temporary runner that
// builds the project's API over infrastructure that never connects,
// captures the YAML it prints, and either writes docs/openapi.yaml
// (then regenerates the typed frontend client) or checks it for
// drift.
package openapi

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	xexec "github.com/dennys-bd/gonext/internal/exec"
	"github.com/dennys-bd/gonext/internal/project"
)

// runnerFilename is the fixed, gitignored name the temp runner is
// materialized under in the target project's backend/.
const runnerFilename = "gonext_openapi_runner.go"

// DocumentPath is where the committed contract lives, relative to
// the project root. The frontend's codegen config reads it from
// there.
const DocumentPath = "docs/openapi.yaml"

const documentFileMode = 0o644

// ErrStale is returned by Check when the committed document no longer
// matches what the backend's registrations produce.
var ErrStale = errors.New("docs/openapi.yaml is stale; run `gonext openapi`")

//go:embed runner_template.go.tmpl
var runnerTemplate []byte

// runFunc executes the materialized runner and returns its stdout;
// overridden in tests to avoid invoking `go run`.
var runFunc = runCapture

// codegenFunc regenerates the frontend client; overridden in tests.
var codegenFunc = xexec.Run

// Generate materializes the runner under root/backend, runs it, and
// returns the OpenAPI document it printed. The runner is removed
// afterward regardless of outcome.
func Generate(ctx context.Context, root string) ([]byte, error) {
	backendDir, remove, err := project.MaterializeRunner(root, runnerFilename, runnerTemplate)
	if err != nil {
		return nil, err
	}
	defer remove()

	doc, err := runFunc(ctx, backendDir, "go", "run", runnerFilename)
	if err != nil {
		return nil, fmt.Errorf("producing openapi document: %w", err)
	}
	return doc, nil
}

// Write regenerates DocumentPath under root and then the typed
// frontend client from it (`pnpm codegen` in frontend/).
func Write(ctx context.Context, root string) error {
	doc, err := Generate(ctx, root)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, DocumentPath), doc, documentFileMode); err != nil {
		return fmt.Errorf("writing %s: %w", DocumentPath, err)
	}
	if err := codegenFunc(ctx, filepath.Join(root, "frontend"), "pnpm", "codegen"); err != nil {
		return fmt.Errorf("regenerating frontend client: %w", err)
	}
	return nil
}

// Check returns ErrStale when DocumentPath under root differs from a
// freshly generated document, so CI can fail on a forgotten regen.
func Check(ctx context.Context, root string) error {
	want, err := Generate(ctx, root)
	if err != nil {
		return err
	}
	got, err := os.ReadFile(filepath.Join(root, DocumentPath))
	if err != nil {
		return fmt.Errorf("reading %s: %w", DocumentPath, err)
	}
	if !bytes.Equal(got, want) {
		return ErrStale
	}
	return nil
}

// runCapture runs name with args in dir, returning its stdout and
// streaming stderr through so build errors stay visible.
func runCapture(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	return cmd.Output()
}
