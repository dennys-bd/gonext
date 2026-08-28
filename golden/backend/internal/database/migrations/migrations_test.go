package migrations

import "testing"

func TestMigrations_RegistersCreateStubs(t *testing.T) {
	sorted := Migrations.Sorted()
	if len(sorted) != 1 {
		t.Fatalf("expected 1 registered migration, got %d", len(sorted))
	}
}
