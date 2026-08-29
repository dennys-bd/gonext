package idgen_test

import (
	"testing"

	"[PROJECT-NAME]/backend/users/internal/idgen"
)

func TestNew_ReturnsDistinctHexIDs(t *testing.T) {
	const iterations = 100

	seen := make(map[string]struct{}, iterations)
	for range iterations {
		id, err := idgen.New()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(id) != 64 {
			t.Fatalf("expected 64 hex chars (256 bits), got %d: %q", len(id), id)
		}
		for _, c := range id {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Fatalf("expected lowercase hex, got %q", id)
			}
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("expected distinct ids, got %q twice", id)
		}
		seen[id] = struct{}{}
	}
}
