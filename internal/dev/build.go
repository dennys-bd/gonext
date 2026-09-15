package dev

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// BinaryPath is where Build writes the compiled dev server binary,
// relative to the project root.
const BinaryPath = "backend/bin/dev-server"

// Build compiles <root>/backend into BinaryPath, streaming compiler output
// to stderr. On failure it leaves any previously built binary untouched.
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
