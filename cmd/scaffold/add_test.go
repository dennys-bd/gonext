package main

import (
	"slices"
	"testing"
)

func TestParseAddAgentArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantNames []string
		wantForce bool
		wantErr   bool
	}{
		{name: "single tool", args: []string{"claude"}, wantNames: []string{"claude"}},
		{name: "flag after names", args: []string{"claude", "cursor", "--force"}, wantNames: []string{"claude", "cursor"}, wantForce: true},
		{name: "flag before names", args: []string{"--force", "gemini"}, wantNames: []string{"gemini"}, wantForce: true},
		{name: "names are not validated here", args: []string{"vim"}, wantNames: []string{"vim"}},
		{name: "no names is a usage error", args: []string{}, wantErr: true},
		{name: "flag only is a usage error", args: []string{"--force"}, wantErr: true},
		{name: "unknown flag errors", args: []string{"claude", "--bogus"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names, force, err := parseAddAgentArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseAddAgentArgs(%v): expected error, got nil", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseAddAgentArgs(%v): unexpected error: %v", tt.args, err)
			}
			if !slices.Equal(names, tt.wantNames) {
				t.Errorf("names = %v, want %v", names, tt.wantNames)
			}
			if force != tt.wantForce {
				t.Errorf("force = %v, want %v", force, tt.wantForce)
			}
		})
	}
}

func TestRunAdd_RejectsUnknownTarget(t *testing.T) {
	if got := runAdd(nil); got != 1 {
		t.Errorf("runAdd(nil) = %d, want 1", got)
	}
	if got := runAdd([]string{"pack"}); got != 1 {
		t.Errorf("runAdd([]string{\"pack\"}) = %d, want 1", got)
	}
}
