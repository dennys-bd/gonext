package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnv_SetsUnsetVariablesOnly(t *testing.T) {
	root := t.TempDir()
	writeEnv(t, root, "# comment\n\nDATABASE_URL=postgres://from-file\nexport ENV=dev\nQUOTED=\"a b\"\nSINGLE='c'\nnoequals\n")
	t.Setenv("ENV", "prod")
	for _, key := range []string{"DATABASE_URL", "QUOTED", "SINGLE"} {
		t.Setenv(key, "") // registers cleanup, then unset so LoadEnv sees them absent
		os.Unsetenv(key)
	}

	if err := LoadEnv(root); err != nil {
		t.Fatalf("LoadEnv: unexpected error: %v", err)
	}

	tests := map[string]string{
		"DATABASE_URL": "postgres://from-file",
		"ENV":          "prod", // already exported: the shell wins over .env
		"QUOTED":       "a b",
		"SINGLE":       "c",
	}
	for key, want := range tests {
		if got := os.Getenv(key); got != want {
			t.Errorf("LoadEnv: %s = %q, want %q", key, got, want)
		}
	}
}

func TestLoadEnv_MissingFileIsNotAnError(t *testing.T) {
	if err := LoadEnv(t.TempDir()); err != nil {
		t.Errorf("LoadEnv: unexpected error without .env: %v", err)
	}
}

func writeEnv(t *testing.T, root, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, envFilename), []byte(content), 0o644); err != nil {
		t.Fatalf("writing .env: %v", err)
	}
}
