package main

import (
	"os"
	"path/filepath"
	"testing"

	gonext "github.com/dennys-bd/gonext"
	"github.com/dennys-bd/gonext/internal/scaffold"
)

// agentRelevantPaths are every path an agent tool owns (see
// internal/scaffold's agentPaths), plus .github/workflows/ci.yml as a
// control that must always be present regardless of selection.
var agentRelevantPaths = []string{
	"CLAUDE.md",
	".claude/settings.json",
	".claude/skills/new-branch/SKILL.md",
	".cursor/rules/gonext.mdc",
	".github/copilot-instructions.md",
	"GEMINI.md",
	".github/workflows/ci.yml",
}

// TestCopy_AgentFileSets pins, per selection, exactly which agent-relevant
// paths a Copy() run writes.
func TestCopy_AgentFileSets(t *testing.T) {
	tests := []struct {
		name    string
		agents  []string
		present []string
		absent  []string
	}{
		{
			name:    "none",
			agents:  nil,
			present: []string{".github/workflows/ci.yml"},
			absent:  []string{"CLAUDE.md", ".claude/settings.json", ".claude/skills/new-branch/SKILL.md", ".cursor/rules/gonext.mdc", ".github/copilot-instructions.md", "GEMINI.md"},
		},
		{
			name:    "claude",
			agents:  []string{"claude"},
			present: []string{".github/workflows/ci.yml", "CLAUDE.md", ".claude/settings.json", ".claude/skills/new-branch/SKILL.md"},
			absent:  []string{".cursor/rules/gonext.mdc", ".github/copilot-instructions.md", "GEMINI.md"},
		},
		{
			name:    "cursor",
			agents:  []string{"cursor"},
			present: []string{".github/workflows/ci.yml", ".cursor/rules/gonext.mdc"},
			absent:  []string{"CLAUDE.md", ".claude/settings.json", ".claude/skills/new-branch/SKILL.md", ".github/copilot-instructions.md", "GEMINI.md"},
		},
		{
			name:    "copilot",
			agents:  []string{"copilot"},
			present: []string{".github/workflows/ci.yml", ".github/copilot-instructions.md"},
			absent:  []string{"CLAUDE.md", ".claude/settings.json", ".claude/skills/new-branch/SKILL.md", ".cursor/rules/gonext.mdc", "GEMINI.md"},
		},
		{
			name:    "gemini",
			agents:  []string{"gemini"},
			present: []string{".github/workflows/ci.yml", "GEMINI.md"},
			absent:  []string{"CLAUDE.md", ".claude/settings.json", ".claude/skills/new-branch/SKILL.md", ".cursor/rules/gonext.mdc", ".github/copilot-instructions.md"},
		},
		{
			name:    "codex",
			agents:  []string{"codex"},
			present: []string{".github/workflows/ci.yml"},
			absent:  []string{"CLAUDE.md", ".claude/settings.json", ".claude/skills/new-branch/SKILL.md", ".cursor/rules/gonext.mdc", ".github/copilot-instructions.md", "GEMINI.md"},
		},
		{
			name:    "claude+cursor",
			agents:  []string{"claude", "cursor"},
			present: []string{".github/workflows/ci.yml", "CLAUDE.md", ".claude/settings.json", ".claude/skills/new-branch/SKILL.md", ".cursor/rules/gonext.mdc"},
			absent:  []string{".github/copilot-instructions.md", "GEMINI.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dest := t.TempDir()
			if err := scaffold.Copy(gonext.Templates, "templates", dest, goldenSlug, tt.agents); err != nil {
				t.Fatalf("Copy: unexpected error: %v", err)
			}

			if _, err := os.Stat(filepath.Join(dest, "AGENTS.md")); err != nil {
				t.Errorf("AGENTS.md: expected unconditional presence: %v", err)
			}
			for _, rel := range tt.present {
				if _, err := os.Stat(filepath.Join(dest, filepath.FromSlash(rel))); err != nil {
					t.Errorf("%s: expected present, got: %v", rel, err)
				}
			}
			for _, rel := range tt.absent {
				if _, err := os.Stat(filepath.Join(dest, filepath.FromSlash(rel))); !os.IsNotExist(err) {
					t.Errorf("%s: expected absent, but exists", rel)
				}
			}
		})
	}
}
