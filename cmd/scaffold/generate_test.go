package main

import "testing"

func TestParseGenerateMigrationArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantDomain string
		wantName   string
		wantErr    bool
	}{
		{name: "domain and name", args: []string{"users", "create_users"}, wantDomain: "users", wantName: "create_users"},
		{name: "missing name", args: []string{"users"}, wantErr: true},
		{name: "no args", args: []string{}, wantErr: true},
		{name: "extra positional", args: []string{"users", "x", "y"}, wantErr: true},
		{name: "unknown flag", args: []string{"users", "x", "--after=users/0001"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domain, name, err := parseGenerateMigrationArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseGenerateMigrationArgs(%v): expected error, got nil", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseGenerateMigrationArgs(%v): unexpected error: %v", tt.args, err)
			}
			if domain != tt.wantDomain {
				t.Errorf("domain = %q, want %q", domain, tt.wantDomain)
			}
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
		})
	}
}

func TestRunGenerate_RejectsUnknownTarget(t *testing.T) {
	if got := runGenerate(nil); got != 1 {
		t.Errorf("runGenerate(nil) = %d, want 1", got)
	}
	if got := runGenerate([]string{"resource"}); got != 1 {
		t.Errorf("runGenerate([]string{\"resource\"}) = %d, want 1", got)
	}
}
