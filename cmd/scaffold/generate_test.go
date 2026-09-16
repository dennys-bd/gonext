package main

import (
	"reflect"
	"testing"

	"github.com/dennys-bd/gonext/internal/generate"
)

func TestParseGenerateArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    generateMode
		wantErr bool
	}{
		{name: "no args", args: nil, want: generateRun},
		{name: "check", args: []string{"--check"}, want: generateCheck},
		{name: "migration", args: []string{"migration", "a", "b"}, want: generateMigration},
		{name: "page", args: []string{"page", "stubs/[id]", "get-stub"}, want: generatePage},
		{name: "resource", args: []string{"resource", "example", "product"}, want: generateResource},
		{name: "unknown flag", args: []string{"--bogus"}, wantErr: true},
		{name: "unknown positional", args: []string{"wire"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGenerateArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseGenerateArgs(%v): expected error, got nil", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseGenerateArgs(%v): unexpected error: %v", tt.args, err)
			}
			if got != tt.want {
				t.Errorf("parseGenerateArgs(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestParseTwoPositionals(t *testing.T) {
	tests := []struct {
		name    string
		usage   string
		args    []string
		wantA   string
		wantB   string
		wantErr bool
	}{
		{name: "migration: domain and name", usage: generateMigrationUsage, args: []string{"users", "create_users"}, wantA: "users", wantB: "create_users"},
		{name: "migration: missing name", usage: generateMigrationUsage, args: []string{"users"}, wantErr: true},
		{name: "migration: no args", usage: generateMigrationUsage, args: []string{}, wantErr: true},
		{name: "migration: extra positional", usage: generateMigrationUsage, args: []string{"users", "x", "y"}, wantErr: true},
		{name: "migration: unknown flag", usage: generateMigrationUsage, args: []string{"users", "x", "--after=users/0001"}, wantErr: true},
		{name: "page: route and operationId", usage: generatePageUsage, args: []string{"stubs/[id]", "get-stub"}, wantA: "stubs/[id]", wantB: "get-stub"},
		{name: "page: missing operationId", usage: generatePageUsage, args: []string{"stubs/[id]"}, wantErr: true},
		{name: "page: --force rejected", usage: generatePageUsage, args: []string{"stubs/[id]", "get-stub", "--force"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, b, err := parseTwoPositionals(tt.args, tt.usage)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseTwoPositionals(%v): expected error, got nil", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseTwoPositionals(%v): unexpected error: %v", tt.args, err)
			}
			if a != tt.wantA {
				t.Errorf("a = %q, want %q", a, tt.wantA)
			}
			if b != tt.wantB {
				t.Errorf("b = %q, want %q", b, tt.wantB)
			}
		})
	}
}

func TestRunGenerate_RejectsUnknownTarget(t *testing.T) {
	if got := runGenerate([]string{"worker"}); got != 1 {
		t.Errorf("runGenerate([]string{\"worker\"}) = %d, want 1", got)
	}
	if got := runGenerate([]string{"--bogus"}); got != 1 {
		t.Errorf("runGenerate([]string{\"--bogus\"}) = %d, want 1", got)
	}
}

func TestParseGenerateResourceArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    resourceArgs
		wantErr string
	}{
		{
			name: "domain, name and fields",
			args: []string{"example", "product", "title:string", "price:int"},
			want: resourceArgs{spec: generate.ResourceSpec{Domain: "example", Name: "product", Fields: []string{"title:string", "price:int"}}},
		},
		{
			name: "--ops with a space",
			args: []string{"example", "product", "--ops", "create,get"},
			want: resourceArgs{spec: generate.ResourceSpec{Domain: "example", Name: "product"}, opsFlag: "create,get", opsSet: true},
		},
		{
			name: "--ops=value",
			args: []string{"example", "product", "--ops=create,get"},
			want: resourceArgs{spec: generate.ResourceSpec{Domain: "example", Name: "product"}, opsFlag: "create,get", opsSet: true},
		},
		{
			name: "--auth public",
			args: []string{"example", "product", "--auth", "public"},
			want: resourceArgs{spec: generate.ResourceSpec{Domain: "example", Name: "product", Auth: "public"}},
		},
		{
			name: "--plural categories",
			args: []string{"example", "category", "--plural", "categories"},
			want: resourceArgs{spec: generate.ResourceSpec{Domain: "example", Name: "category", Plural: "categories"}},
		},
		{
			name: "flags after positionals",
			args: []string{"example", "product", "title:string", "--ops", "create", "--auth", "public"},
			want: resourceArgs{spec: generate.ResourceSpec{Domain: "example", Name: "product", Fields: []string{"title:string"}, Auth: "public"}, opsFlag: "create", opsSet: true},
		},
		{
			name: "flag between positionals",
			args: []string{"example", "--ops", "create", "product"},
			want: resourceArgs{spec: generate.ResourceSpec{Domain: "example", Name: "product"}, opsFlag: "create", opsSet: true},
		},
		{
			name:    "unknown flag",
			args:    []string{"example", "product", "--bogus"},
			wantErr: `unknown flag "--bogus"`,
		},
		{
			name:    "--ops with no value",
			args:    []string{"example", "product", "--ops"},
			wantErr: "flag --ops needs a value",
		},
		{
			name:    "single positional",
			args:    []string{"example"},
			wantErr: generateResourceUsage,
		},
		{
			name:    "no positionals",
			args:    []string{"--ops", "create"},
			wantErr: generateResourceUsage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGenerateResourceArgs(tt.args)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("parseGenerateResourceArgs(%v): err = %v, want %q", tt.args, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseGenerateResourceArgs(%v): unexpected error: %v", tt.args, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseGenerateResourceArgs(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestResolveOps(t *testing.T) {
	promptCalled := false
	prompt := func(all []string) ([]string, error) {
		promptCalled = true
		return []string{"create"}, nil
	}

	t.Run("flag set", func(t *testing.T) {
		promptCalled = false
		got, err := resolveOps("create,get", true, false, prompt)
		if err != nil {
			t.Fatalf("resolveOps: unexpected error: %v", err)
		}
		if want := []string{"create", "get"}; !reflect.DeepEqual(got, want) {
			t.Errorf("resolveOps = %v, want %v", got, want)
		}
		if promptCalled {
			t.Error("resolveOps: prompt called although --ops was set")
		}
	})

	t.Run("no flag, not a TTY", func(t *testing.T) {
		promptCalled = false
		got, err := resolveOps("", false, false, prompt)
		if err != nil {
			t.Fatalf("resolveOps: unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("resolveOps = %v, want nil", got)
		}
		if promptCalled {
			t.Error("resolveOps: prompt called without a TTY")
		}
	})

	t.Run("no flag, TTY", func(t *testing.T) {
		promptCalled = false
		got, err := resolveOps("", false, true, prompt)
		if err != nil {
			t.Fatalf("resolveOps: unexpected error: %v", err)
		}
		if want := []string{"create"}; !reflect.DeepEqual(got, want) {
			t.Errorf("resolveOps = %v, want %v", got, want)
		}
		if !promptCalled {
			t.Error("resolveOps: prompt not called on a TTY")
		}
	})
}
