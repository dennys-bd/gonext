package main

import "testing"

func TestParseMigrateArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantTarget string
		wantYes    bool
		wantErr    bool
	}{
		{name: "no args applies everything", args: []string{}},
		{name: "target only", args: []string{"users/0002"}, wantTarget: "users/0002"},
		{name: "target and yes", args: []string{"users/0002", "--yes"}, wantTarget: "users/0002", wantYes: true},
		{name: "yes before target", args: []string{"--yes", "users/zero"}, wantTarget: "users/zero", wantYes: true},
		{name: "yes only, no target", args: []string{"--yes"}, wantYes: true},
		{name: "two positionals is a usage error", args: []string{"users/0002", "orders/0001"}, wantErr: true},
		{name: "unknown flag errors", args: []string{"--bogus"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, yes, err := parseMigrateArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseMigrateArgs(%v): expected error, got nil", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseMigrateArgs(%v): unexpected error: %v", tt.args, err)
			}
			if target != tt.wantTarget {
				t.Errorf("target = %q, want %q", target, tt.wantTarget)
			}
			if yes != tt.wantYes {
				t.Errorf("yes = %v, want %v", yes, tt.wantYes)
			}
		})
	}
}
