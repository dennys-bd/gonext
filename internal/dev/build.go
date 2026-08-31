package dev

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// BinaryPath is where Build writes the compiled dev server binary,
// relative to the project root. It reuses the same gitignored
// backend/bin/ convention `make smoke` already writes its own
// temporary binary to.
const BinaryPath = "backend/bin/dev-server"

// Build compiles <root>/backend into BinaryPath, streaming compiler
// output to stderr. On failure it returns an error and leaves any
// previously built binary untouched — it never removes or truncates
// the output path itself, `go build -o` only overwrites it on a
// successful compile.
func Build(ctx context.Context, root string) error {
	out := filepath.Join(root, BinaryPath)
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(out), err)
	}

	cmd := exec.CommandContext(ctx, "go", "build", "-o", out, "./backend")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build ./backend: %w", err)
	}
	return nil
}
