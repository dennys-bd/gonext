package main

import "testing"

func TestParseInitArgs(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantName      string
		wantPath      string
		wantAgents    string
		wantAgentsSet bool
		wantErr       bool
	}{
		{name: "name only", args: []string{"my-app"}, wantName: "my-app"},
		{name: "name and path", args: []string{"my-app", "./x"}, wantName: "my-app", wantPath: "./x"},
		{
			name:          "flag after positionals",
			args:          []string{"my-app", "--agents=claude,cursor"},
			wantName:      "my-app",
			wantAgents:    "claude,cursor",
			wantAgentsSet: true,
		},
		{
			name:          "flag before positionals",
			args:          []string{"--agents=none", "my-app"},
			wantName:      "my-app",
			wantAgents:    "none",
			wantAgentsSet: true,
		},
		{name: "unknown flag errors", args: []string{"my-app", "--bogus"}, wantErr: true},
		{name: "malformed agents flag errors", args: []string{"--agents"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, path, agents, agentsSet, err := parseInitArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseInitArgs(%v): expected error, got nil", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseInitArgs(%v): unexpected error: %v", tt.args, err)
			}
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
			if path != tt.wantPath {
				t.Errorf("path = %q, want %q", path, tt.wantPath)
			}
			if agents != tt.wantAgents {
				t.Errorf("agents = %q, want %q", agents, tt.wantAgents)
			}
			if agentsSet != tt.wantAgentsSet {
				t.Errorf("agentsSet = %v, want %v", agentsSet, tt.wantAgentsSet)
			}
		})
	}
}
