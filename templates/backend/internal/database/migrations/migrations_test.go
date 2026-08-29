package migrations

import "testing"

func TestMigrations_RegistersCreateStubs(t *testing.T) {
	sorted := Migrations.Sorted()
	if len(sorted) != 2 {
		t.Fatalf("expected 2 registered migrations, got %d", len(sorted))
	}
}
