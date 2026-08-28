package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestInit_E2E runs a real `gonext init` into a temp directory and
// asserts the generated project builds and installs its frontend
// dependencies. It hits the network and can take a while, so it is
// opt-in only: run with GONEXT_E2E=1, as its own CI job, mirroring
// the templates' own `make smoke` (slow, Docker-backed) split from
// `make test` (fast).
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

	pnpmCmd := exec.Command("pnpm", "install", "--frozen-lockfile")
	pnpmCmd.Dir = filepath.Join(dest, "frontend")
	if out, err := pnpmCmd.CombinedOutput(); err != nil {
		t.Errorf("pnpm install --frozen-lockfile failed in generated project: %v\n%s", err, out)
	}
}
