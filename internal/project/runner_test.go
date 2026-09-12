package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaterializeRunner_SubstitutesModulePathAndRemoves(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, root, "example.com/foo")
	backendDir := filepath.Join(root, "backend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("creating backend dir: %v", err)
	}
	tmpl := []byte("import \"" + ModulePathToken + "/backend/internal/config\"\n")

	gotDir, remove, err := MaterializeRunner(root, "gonext_test_runner.go", tmpl, nil)
	if err != nil {
		t.Fatalf("MaterializeRunner: unexpected error: %v", err)
	}
	if gotDir != backendDir {
		t.Errorf("MaterializeRunner: dir = %q, want %q", gotDir, backendDir)
	}

	runnerPath := filepath.Join(backendDir, "gonext_test_runner.go")
	data, err := os.ReadFile(runnerPath)
	if err != nil {
		t.Fatalf("reading materialized runner: %v", err)
	}
	if got := string(data); !strings.Contains(got, "example.com/foo/backend/internal/config") || strings.Contains(got, ModulePathToken) {
		t.Errorf("MaterializeRunner: token not substituted, got %q", got)
	}

	remove()
	if _, err := os.Stat(runnerPath); !os.IsNotExist(err) {
		t.Errorf("MaterializeRunner: expected runner removed, stat err = %v", err)
	}
}

func TestMaterializeRunner_ErrorsWhenModuleNotFound(t *testing.T) {
	if _, _, err := MaterializeRunner(t.TempDir(), "gonext_test_runner.go", nil, nil); err == nil {
		t.Errorf("MaterializeRunner: expected error when go.mod missing, got nil")
	}
}

// extra tokens are substituted before ModulePathToken, so their values
// may themselves contain ModulePathToken.
func TestMaterializeRunner_SubstitutesExtraTokens(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, root, "example.com/foo")
	backendDir := filepath.Join(root, "backend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("creating backend dir: %v", err)
	}
	tmpl := []byte("[X]\n" + ModulePathToken + "\n")
	extra := map[string]string{"[X]": "_ \"" + ModulePathToken + "/backend/users/migrations\""}

	_, remove, err := MaterializeRunner(root, "gonext_test_runner.go", tmpl, extra)
	if err != nil {
		t.Fatalf("MaterializeRunner: unexpected error: %v", err)
	}
	defer remove()

	data, err := os.ReadFile(filepath.Join(backendDir, "gonext_test_runner.go"))
	if err != nil {
		t.Fatalf("reading materialized runner: %v", err)
	}
	want := "_ \"example.com/foo/backend/users/migrations\"\nexample.com/foo\n"
	if got := string(data); got != want {
		t.Errorf("MaterializeRunner: content = %q, want %q", got, want)
	}
}
