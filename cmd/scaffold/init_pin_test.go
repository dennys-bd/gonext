package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dennys-bd/gonext/internal/scaffold"
)

// A scaffolded project must pin the auth contract to the version this
// CLI was built against rather than resolving `latest`: a project
// generated today has to keep building after the contract ships a
// v0.2.0.
func TestPinAuthModule(t *testing.T) {
	dir := t.TempDir()
	gomod := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(gomod, []byte("module example\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := pinAuthModule(dir); err != nil {
		t.Fatalf("pinAuthModule: %v", err)
	}

	data, err := os.ReadFile(gomod)
	if err != nil {
		t.Fatal(err)
	}
	want := scaffold.AuthModulePath + " " + scaffold.AuthModuleVersion
	if !strings.Contains(string(data), want) {
		t.Errorf("expected go.mod to require %q, got:\n%s", want, data)
	}
}
