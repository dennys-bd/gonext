package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/dennys-bd/gonext/internal/scaffold"
)

// A scaffolded project must pin gonext to how this CLI was built:
// a published version resolves `go mod tidy` to that exact tag rather than
// `latest`, and a checkout build points at the working tree instead of a
// pseudo-version nobody published.
func TestPinGonextModule(t *testing.T) {
	tests := map[string]struct {
		version string
		want    string
	}{
		"published version": {
			version: "v0.3.0",
			want:    scaffold.ModulePath + " v0.3.0",
		},
		"checkout build": {
			version: "(devel)",
			want:    "replace " + scaffold.ModulePath + " =>",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			restore := stubModuleVersion(t, tc.version)
			defer restore()

			dir := t.TempDir()
			gomod := filepath.Join(dir, "go.mod")
			if err := os.WriteFile(gomod, []byte("module example\n\ngo 1.26\n"), 0o644); err != nil {
				t.Fatal(err)
			}

			if err := pinGonextModule(dir); err != nil {
				t.Fatalf("pinGonextModule: %v", err)
			}

			data, err := os.ReadFile(gomod)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), tc.want) {
				t.Errorf("expected go.mod to contain %q, got:\n%s", tc.want, data)
			}

			if out, err := exec.Command("go", "mod", "edit", "-json", gomod).CombinedOutput(); err != nil {
				t.Errorf("resulting go.mod is not valid: %v\n%s", err, out)
			}
		})
	}
}

// stubModuleVersion makes ModuleEdit see a binary built with the given
// runtime/debug main module version and no vcs.* build setting.
func stubModuleVersion(t *testing.T, version string) func() {
	t.Helper()
	orig := scaffold.ReadBuildInfo
	scaffold.ReadBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: version}}, true
	}
	return func() { scaffold.ReadBuildInfo = orig }
}
