package main

import "testing"

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
	if got := runGenerate([]string{"resource"}); got != 1 {
		t.Errorf("runGenerate([]string{\"resource\"}) = %d, want 1", got)
	}
	if got := runGenerate([]string{"--bogus"}); got != 1 {
		t.Errorf("runGenerate([]string{\"--bogus\"}) = %d, want 1", got)
	}
}
