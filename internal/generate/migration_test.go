package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigration_FirstIsZeroOne(t *testing.T) {
	root := projectRoot(t, "users")

	rel, err := Migration(root, "users", "create_users")
	if err != nil {
		t.Fatalf("Migration: unexpected error: %v", err)
	}
	want := "backend/users/migrations/0001_create_users.go"
	if rel != want {
		t.Errorf("Migration: rel = %q, want %q", rel, want)
	}

	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	content := string(data)
	for _, want := range []string{"package migrations", `"github.com/dennys-bd/gonext/dbmigrate"`, "dbmigrate.Register("} {
		if !strings.Contains(content, want) {
			t.Errorf("Migration: content missing %q, got:\n%s", want, content)
		}
	}
}

func TestMigration_NextAfterGap(t *testing.T) {
	root := projectRoot(t, "users")
	migrationsDir := filepath.Join(root, "backend", "users", "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("creating migrations dir: %v", err)
	}
	for _, name := range []string{"0001_a.go", "0003_b.go"} {
		if err := os.WriteFile(filepath.Join(migrationsDir, name), []byte("package migrations\n"), 0o644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}

	rel, err := Migration(root, "users", "c")
	if err != nil {
		t.Fatalf("Migration: unexpected error: %v", err)
	}
	want := "backend/users/migrations/0004_c.go"
	if rel != want {
		t.Errorf("Migration: rel = %q, want %q", rel, want)
	}
}

func TestMigration_Errors(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		migName string
		wantErr string
	}{
		{"unknown domain", "orders", "create_x", "no such domain backend/orders"},
		{"internal is not a domain", "internal", "create_x", "no such domain backend/internal"},
		{"bad name", "users", "Create-Orders", `invalid migration name "Create-Orders": use snake_case`},
		{"traversal is not a domain", "../frontend", "create_x", "no such domain backend/../frontend"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := projectRoot(t, "users", "internal")
			if err := os.MkdirAll(filepath.Join(root, "frontend"), 0o755); err != nil {
				t.Fatalf("setup: %v", err)
			}

			_, err := Migration(root, tt.domain, tt.migName)
			if err == nil {
				t.Fatalf("Migration: expected error, got nil")
			}
			if err.Error() != tt.wantErr {
				t.Errorf("Migration: err = %q, want %q", err.Error(), tt.wantErr)
			}
			if _, statErr := os.Stat(filepath.Join(root, "backend", tt.domain, "migrations")); !os.IsNotExist(statErr) {
				t.Errorf("Migration: expected backend/%s/migrations not to exist, stat err = %v", tt.domain, statErr)
			}
		})
	}
}

// projectRoot builds a temp project root with a go.mod and an empty
// backend/<domain>/ directory per domain.
func projectRoot(t *testing.T, domains ...string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}
	for _, domain := range domains {
		if err := os.MkdirAll(filepath.Join(root, "backend", domain), 0o755); err != nil {
			t.Fatalf("creating backend/%s: %v", domain, err)
		}
	}
	return root
}

func TestWriteMigration_WritesBodyVerbatim(t *testing.T) {
	root := projectRoot(t, "users")
	body := "package migrations\n// custom\n"

	rel, err := writeMigration(root, "users", "create_things", body)
	if err != nil {
		t.Fatalf("writeMigration: unexpected error: %v", err)
	}
	want := "backend/users/migrations/0001_create_things.go"
	if rel != want {
		t.Errorf("writeMigration: rel = %q, want %q", rel, want)
	}

	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	if string(data) != body {
		t.Errorf("writeMigration: content = %q, want %q", data, body)
	}
}
