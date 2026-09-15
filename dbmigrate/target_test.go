package dbmigrate

import (
	"slices"
	"testing"
)

func TestParseTarget(t *testing.T) {
	tests := []struct {
		in      string
		want    Target
		wantErr bool
	}{
		{in: "users/0002", want: Target{Domain: "users", Version: "0002"}},
		{in: "users/zero", want: Target{Domain: "users", Version: "zero"}},
		{in: "users", wantErr: true},
		{in: "users/0002/x", wantErr: true},
		{in: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseTarget(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseTarget(%q): expected error, got nil", tt.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseTarget(%q): unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("ParseTarget(%q) = %+v, want %+v", tt.in, got, tt.want)
			}
		})
	}
}

// planRegistry builds the registry TestRegistry_Plan's table exercises.
func planRegistry(t *testing.T) *Registry {
	t.Helper()
	r := NewRegistry()
	noop := noopMigrationFunc
	mustAdd(t, r, Migration{Domain: "users", Version: "0001", Name: "x"}, noop, noop)
	mustAdd(t, r, Migration{Domain: "users", Version: "0002", Name: "x"}, noop, noop)
	mustAdd(t, r, Migration{Domain: "users", Version: "0003", Name: "x"}, noop, noop)
	mustAdd(t, r, Migration{Domain: "orders", Version: "0001", Name: "x"}, noop, noop, Dependency{Domain: "users", Version: "0002"})
	mustAdd(t, r, Migration{Domain: "orders", Version: "0002", Name: "x"}, noop, noop)
	mustAdd(t, r, Migration{Domain: "example", Version: "0001", Name: "x"}, noop, noop)
	return r
}

func TestRegistry_Plan(t *testing.T) {
	allApplied := map[string]bool{
		"users/0001": true, "users/0002": true, "users/0003": true,
		"orders/0001": true, "orders/0002": true, "example/0001": true,
	}

	tests := []struct {
		name         string
		applied      map[string]bool
		target       Target
		wantRollback []string
		wantApply    []string
		wantErr      string
	}{
		{
			name:         "rollback to an earlier version pulls dependents first",
			applied:      allApplied,
			target:       Target{Domain: "users", Version: "0001"},
			wantRollback: []string{"orders/0002", "orders/0001", "users/0003", "users/0002"},
		},
		{
			name:         "rollback to zero unwinds the whole domain, other domains untouched",
			applied:      allApplied,
			target:       Target{Domain: "users", Version: "zero"},
			wantRollback: []string{"orders/0002", "orders/0001", "users/0003", "users/0002", "users/0001"},
		},
		{
			name:      "forward from nothing applies only the target's closure",
			applied:   map[string]bool{},
			target:    Target{Domain: "orders", Version: "0002"},
			wantApply: []string{"users/0001", "users/0002", "orders/0001", "orders/0002"},
		},
		{
			name:    "already at target",
			applied: allApplied,
			target:  Target{Domain: "users", Version: "0003"},
		},
		{
			name:    "unregistered version errors",
			applied: allApplied,
			target:  Target{Domain: "users", Version: "0009"},
			wantErr: "users/0009 is not a registered migration",
		},
		{
			name:    "unregistered domain errors",
			applied: allApplied,
			target:  Target{Domain: "nope", Version: "0001"},
			wantErr: "no such domain nope",
		},
		{
			name:    "already at a non-terminal target",
			applied: map[string]bool{"users/0001": true, "users/0002": true},
			target:  Target{Domain: "users", Version: "0002"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := planRegistry(t)
			rollback, apply, err := r.Plan(tt.target, tt.applied)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("Plan(): expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("Plan(): error = %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Plan(): unexpected error: %v", err)
			}

			gotRollback := make([]string, len(rollback))
			for i, m := range rollback {
				gotRollback[i] = m.String()
			}
			if !slices.Equal(gotRollback, tt.wantRollback) {
				t.Errorf("rollback = %v, want %v", gotRollback, tt.wantRollback)
			}

			gotApply := make([]string, len(apply))
			for i, m := range apply {
				gotApply[i] = m.String()
			}
			if !slices.Equal(gotApply, tt.wantApply) {
				t.Errorf("apply = %v, want %v", gotApply, tt.wantApply)
			}
		})
	}
}
