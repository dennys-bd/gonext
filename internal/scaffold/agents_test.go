package scaffold

import (
	"io/fs"
	"path"
	"slices"
	"strings"
	"testing"

	gonext "github.com/dennys-bd/gonext"
)

func TestParseAgents(t *testing.T) {
	tests := []struct {
		name          string
		list          string
		want          []string
		wantErr       bool
		listsAllNames bool
	}{
		{name: "empty", list: "", want: nil},
		{name: "none", list: "none", want: nil},
		{name: "single", list: "claude", want: []string{"claude"}},
		{name: "dedupes and sorts", list: "cursor,claude,cursor", want: []string{"claude", "cursor"}},
		{name: "trims whitespace", list: "claude, gemini ", want: []string{"claude", "gemini"}},
		{name: "codex is a recognised no-op tool", list: "codex", want: []string{"codex"}},
		{name: "unknown name errors", list: "vim", wantErr: true, listsAllNames: true},
		{name: "none mixed with a name errors", list: "none,claude", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAgents(tt.list)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseAgents(%q): expected error, got nil", tt.list)
				}
				if !tt.listsAllNames {
					return
				}
				for _, name := range AgentNames() {
					if !strings.Contains(err.Error(), name) {
						t.Errorf("ParseAgents(%q) error %q does not mention valid name %q", tt.list, err, name)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseAgents(%q): unexpected error: %v", tt.list, err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("ParseAgents(%q) = %v, want %v", tt.list, got, tt.want)
			}
		})
	}
}

func TestParseAgentNames(t *testing.T) {
	tests := []struct {
		name          string
		names         []string
		want          []string
		wantErr       bool
		listsAllNames bool
	}{
		{name: "single", names: []string{"claude"}, want: []string{"claude"}},
		{name: "dedupes and sorts", names: []string{"cursor", "claude", "cursor"}, want: []string{"claude", "cursor"}},
		{name: "codex", names: []string{"codex"}, want: []string{"codex"}},
		{name: "empty", names: nil, want: nil},
		{name: "none is unknown", names: []string{"none"}, wantErr: true, listsAllNames: true},
		{name: "unknown name errors", names: []string{"vim"}, wantErr: true, listsAllNames: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAgentNames(tt.names)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseAgentNames(%v): expected error, got nil", tt.names)
				}
				if !tt.listsAllNames {
					return
				}
				for _, name := range AgentNames() {
					if !strings.Contains(err.Error(), name) {
						t.Errorf("ParseAgentNames(%v) error %q does not mention valid name %q", tt.names, err, name)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseAgentNames(%v): unexpected error: %v", tt.names, err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("ParseAgentNames(%v) = %v, want %v", tt.names, got, tt.want)
			}
		})
	}
}

// TestAgentPaths_ExistInTemplates guards agentPaths against drifting
// from the tree: a path renamed in templates/ without the map being
// updated would otherwise become silently unconditional.
func TestAgentPaths_ExistInTemplates(t *testing.T) {
	for agent, paths := range agentPaths {
		for _, p := range paths {
			t.Run(agent+"/"+p, func(t *testing.T) {
				if _, err := fs.Stat(gonext.Templates, path.Join("templates", p)); err != nil {
					t.Errorf("agentPaths[%q] path %q: %v", agent, p, err)
				}
			})
		}
	}
}
