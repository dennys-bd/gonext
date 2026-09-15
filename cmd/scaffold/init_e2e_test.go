package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/dennys-bd/gonext/internal/migrate"
)

// TestInit_E2E runs a real `gonext init` into a temp directory and asserts
// the generated project builds and installs its frontend dependencies. It
// hits the network, so it's opt-in only: run with GONEXT_E2E=1.
func TestInit_E2E(t *testing.T) {
	if os.Getenv("GONEXT_E2E") == "" {
		t.Skip("set GONEXT_E2E=1 to run the opt-in end-to-end scaffolding test")
	}

	dir := t.TempDir()
	dest := filepath.Join(dir, "e2e-app")

	if code := runInit([]string{"e2e-app", dest}); code != 0 {
		t.Fatalf("runInit: expected exit code 0, got %d", code)
	}

	buildCmd := exec.Command("go", "build", "./...")
	buildCmd.Dir = dest
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Errorf("go build ./... failed in generated project: %v\n%s", err, out)
	}

	// The only place the migration runner template gets compiled against
	// the pinned gonext library — `gonext migrate` itself never runs here
	// since there's no database, but `go vet` proves it still compiles.
	if err := migrate.Vet(context.Background(), dest); err != nil {
		t.Errorf("migrate.Vet failed in generated project: %v", err)
	}

	// Proves the `go get -tool` step landed the tool directive and
	// the committed wire_gen.go files are fresh — the only place this
	// runs, since it needs the real wire binary and go.mod.
	wireCmd := exec.Command("go", "tool", "wire", "diff", "./backend/...")
	wireCmd.Dir = dest
	if out, err := wireCmd.CombinedOutput(); err != nil {
		t.Errorf("go tool wire diff ./backend/... failed in generated project: %v\n%s", err, out)
	}

	pnpmCmd := exec.Command("pnpm", "install", "--frozen-lockfile")
	pnpmCmd.Dir = filepath.Join(dest, "frontend")
	if out, err := pnpmCmd.CombinedOutput(); err != nil {
		t.Errorf("pnpm install --frozen-lockfile failed in generated project: %v\n%s", err, out)
	}
}
