package dbmigrate

import (
	"context"
	"slices"
	"testing"

	"github.com/uptrace/bun"
)

func noopMigrationFunc(ctx context.Context, db *bun.DB) error { return nil }

func TestRegistry_Order(t *testing.T) {
	tests := []struct {
		name string
		add  []struct {
			dom, ver string
			deps     []Dependency
		}
		want    []string
		wantErr string
	}{
		{
			name: "single domain orders by version",
			add: []struct {
				dom, ver string
				deps     []Dependency
			}{
				{"users", "0002", nil},
				{"users", "0001", nil},
			},
			want: []string{"users/0001", "users/0002"},
		},
		{
			name: "independent domains interleave by key",
			add: []struct {
				dom, ver string
				deps     []Dependency
			}{
				{"users", "0001", nil},
				{"users", "0002", nil},
				{"example", "0001", nil},
			},
			want: []string{"example/0001", "users/0001", "users/0002"},
		},
		{
			name: "After pulls a later-sorting domain ahead",
			add: []struct {
				dom, ver string
				deps     []Dependency
			}{
				{"example", "0001", []Dependency{{"users", "0001"}}},
				{"users", "0001", nil},
			},
			want: []string{"users/0001", "example/0001"},
		},
		{
			name: "After on a later version pulls the whole chain",
			add: []struct {
				dom, ver string
				deps     []Dependency
			}{
				{"a", "0001", []Dependency{{"z", "0002"}}},
				{"z", "0001", nil},
				{"z", "0002", nil},
			},
			want: []string{"z/0001", "z/0002", "a/0001"},
		},
		{
			name: "cycle",
			add: []struct {
				dom, ver string
				deps     []Dependency
			}{
				{"users", "0003", []Dependency{{"orders", "0002"}}},
				{"orders", "0002", []Dependency{{"users", "0003"}}},
			},
			wantErr: "migration dependency cycle: orders/0002 -> users/0003 -> orders/0002",
		},
		{
			name: "unregistered target",
			add: []struct {
				dom, ver string
				deps     []Dependency
			}{
				{"users", "0003", []Dependency{{"orders", "0002"}}},
			},
			wantErr: "migration users/0003 depends on orders/0002, which is not registered",
		},
		{
			name: "empty registry",
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			noop := noopMigrationFunc
			for _, a := range tt.add {
				m := Migration{Domain: a.dom, Version: a.ver, Name: "x"}
				if err := r.Add(m, noop, noop, a.deps...); err != nil {
					t.Fatalf("Add(%s): unexpected error: %v", m, err)
				}
			}

			order, err := r.Order()
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("Order(): expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("Order(): error = %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Order(): unexpected error: %v", err)
			}

			got := make([]string, len(order))
			for i, m := range order {
				got[i] = m.String()
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Order() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegistry_Add_RejectsDuplicate(t *testing.T) {
	r := NewRegistry()
	noop := noopMigrationFunc
	m := Migration{Domain: "users", Version: "0001", Name: "x"}
	if err := r.Add(m, noop, noop); err != nil {
		t.Fatalf("Add: unexpected error: %v", err)
	}
	err := r.Add(m, noop, noop)
	if err == nil {
		t.Fatalf("Add: expected error on duplicate, got nil")
	}
	want := "migration users/0001 registered twice"
	if err.Error() != want {
		t.Errorf("Add: error = %q, want %q", err.Error(), want)
	}
}

func TestParsePath(t *testing.T) {
	tests := []struct {
		file    string
		want    Migration
		wantErr bool
	}{
		{
			file: "/home/x/proj/backend/users/migrations/0002_create_users.go",
			want: Migration{Domain: "users", Version: "0002", Name: "create_users"},
		},
		{
			file: "backend/users/migrations/0002_create_users.go",
			want: Migration{Domain: "users", Version: "0002", Name: "create_users"},
		},
		{
			file:    "/x/backend/internal/database/migrations/0001_create_stubs.go",
			wantErr: true,
		},
		{
			file:    "/x/backend/users/migrations/2_x.go",
			wantErr: true,
		},
		{
			file:    "/x/backend/users/migrations/0001_Create.go",
			wantErr: true,
		},
		{
			file:    "/x/backend/users/other/0001_x.go",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			got, err := parsePath(tt.file)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parsePath(%q): expected error, got nil", tt.file)
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePath(%q): unexpected error: %v", tt.file, err)
			}
			if got != tt.want {
				t.Errorf("parsePath(%q) = %+v, want %+v", tt.file, got, tt.want)
			}
		})
	}
}
