package scaffold

import (
	"bytes"
	"embed"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	gonext "github.com/dennys-bd/gonext"
)

//go:embed all:testdata/fixture
var fixtureFS embed.FS

func TestCopy_SubstitutesTextFiles(t *testing.T) {
	dest := t.TempDir()

	if err := Copy(fixtureFS, "testdata/fixture", dest, "my-app", nil); err != nil {
		t.Fatalf("Copy: unexpected error: %v", err)
	}

	readme, err := os.ReadFile(filepath.Join(dest, "README.md"))
	if err != nil {
		t.Fatalf("reading README.md: %v", err)
	}
	want := "# my-app\n\nWelcome to my-app! This is the my-app README.\n"
	if string(readme) != want {
		t.Errorf("README.md = %q, want %q", readme, want)
	}

	gomod, err := os.ReadFile(filepath.Join(dest, "go.mod.txt"))
	if err != nil {
		t.Fatalf("reading go.mod.txt: %v", err)
	}
	if strings.Contains(string(gomod), "[PROJECT-NAME]") {
		t.Errorf("go.mod.txt still contains [PROJECT-NAME] token: %q", gomod)
	}
	if !strings.Contains(string(gomod), "module my-app") {
		t.Errorf("go.mod.txt missing substituted module line: %q", gomod)
	}

	tracksGo, err := os.ReadFile(filepath.Join(dest, "sub", "tracks.go"))
	if err != nil {
		t.Fatalf("reading sub/tracks.go: %v", err)
	}
	if strings.Contains(string(tracksGo), "[PROJECT-NAME]") {
		t.Errorf("sub/tracks.go still contains [PROJECT-NAME] token: %q", tracksGo)
	}
}

func TestCopy_NoTokenSurvives(t *testing.T) {
	dest := t.TempDir()

	if err := Copy(fixtureFS, "testdata/fixture", dest, "my-app", nil); err != nil {
		t.Fatalf("Copy: unexpected error: %v", err)
	}

	err := filepath.Walk(dest, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if strings.HasSuffix(path, "icon.bin") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if bytes.Contains(data, []byte("[PROJECT-NAME]")) {
			t.Errorf("%s still contains [PROJECT-NAME] token", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking dest: %v", err)
	}
}

func TestCopy_BinaryFileUnchanged(t *testing.T) {
	dest := t.TempDir()

	if err := Copy(fixtureFS, "testdata/fixture", dest, "my-app", nil); err != nil {
		t.Fatalf("Copy: unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dest, "sub", "icon.bin"))
	if err != nil {
		t.Fatalf("reading sub/icon.bin: %v", err)
	}
	want, err := fixtureFS.ReadFile("testdata/fixture/sub/icon.bin")
	if err != nil {
		t.Fatalf("reading fixture icon.bin: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("binary file was modified: got %v, want %v", got, want)
	}
}

func TestCopy_WritesWritableFiles(t *testing.T) {
	dest := t.TempDir()

	if err := Copy(fixtureFS, "testdata/fixture", dest, "my-app", nil); err != nil {
		t.Fatalf("Copy: unexpected error: %v", err)
	}

	info, err := os.Stat(filepath.Join(dest, "README.md"))
	if err != nil {
		t.Fatalf("stat README.md: %v", err)
	}
	if info.Mode().Perm()&0o200 == 0 {
		t.Errorf("README.md mode = %v, want owner-writable (embed.FS files are always read-only; Copy must not propagate that mode to its output)", info.Mode().Perm())
	}
}

func TestCopy_SkipsGoModAndGoSum(t *testing.T) {
	// go:embed treats a directory containing go.mod as a separate module and
	// won't embed it, so os.DirFS is used here instead.
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "go.mod"), []byte("module should-not-be-copied\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "go.sum"), []byte("should-not-be-copied\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	dest := t.TempDir()
	if err := Copy(os.DirFS(src), ".", dest, "my-app", nil); err != nil {
		t.Fatalf("Copy: unexpected error: %v", err)
	}

	for _, name := range []string{"go.mod", "go.sum"} {
		if _, err := os.Stat(filepath.Join(dest, name)); !os.IsNotExist(err) {
			t.Errorf("expected %s to be skipped by Copy, but it exists at %s", name, dest)
		}
	}
	if _, err := os.Stat(filepath.Join(dest, "main.go")); err != nil {
		t.Errorf("expected main.go to still be copied: %v", err)
	}
}

func TestCopy_SkipsUnselectedAgentPaths(t *testing.T) {
	src := t.TempDir()
	files := map[string]string{
		"AGENTS.md":                       "guardrails\n",
		"CLAUDE.md":                       "pointer\n",
		".claude/settings.json":           "{}\n",
		".github/workflows/ci.yml":        "on: push\n",
		".github/copilot-instructions.md": "pointer\n",
	}
	for rel, content := range files {
		full := filepath.Join(src, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("setup: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	dest := t.TempDir()
	if err := Copy(os.DirFS(src), ".", dest, "my-app", []string{"copilot"}); err != nil {
		t.Fatalf("Copy: unexpected error: %v", err)
	}

	for _, rel := range []string{"AGENTS.md", ".github/workflows/ci.yml", ".github/copilot-instructions.md"} {
		if _, err := os.Stat(filepath.Join(dest, filepath.FromSlash(rel))); err != nil {
			t.Errorf("expected %s to be copied: %v", rel, err)
		}
	}
	for _, rel := range []string{"CLAUDE.md", ".claude"} {
		if _, err := os.Stat(filepath.Join(dest, filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Errorf("expected %s to be skipped by Copy, but it exists at %s", rel, dest)
		}
	}
}

// scaffoldDest returns a real default scaffold (no tool selected) in a
// fresh t.TempDir(): it has AGENTS.md and .github/workflows/ci.yml,
// but no tool files, matching what `gonext init` writes today.
func scaffoldDest(t *testing.T) string {
	t.Helper()
	dest := t.TempDir()
	if err := Copy(gonext.Templates, "templates", dest, "my-app", nil); err != nil {
		t.Fatalf("Copy: unexpected error: %v", err)
	}
	return dest
}

func TestAddAgents_WritesOnlyOwnedPaths(t *testing.T) {
	dest := scaffoldDest(t)

	written, err := AddAgents(gonext.Templates, "templates", dest, "my-app", []string{"claude", "copilot"}, false)
	if err != nil {
		t.Fatalf("AddAgents: unexpected error: %v", err)
	}

	want := []string{"CLAUDE.md", ".claude/settings.json", ".claude/skills/new-branch/SKILL.md", ".github/copilot-instructions.md"}
	if !slices.Equal(written, want) {
		t.Errorf("AddAgents written = %v, want %v", written, want)
	}

	claude, err := os.ReadFile(filepath.Join(dest, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("reading CLAUDE.md: %v", err)
	}
	if !strings.HasPrefix(string(claude), "# my-app\n") {
		t.Errorf("CLAUDE.md = %q, want prefix %q", claude, "# my-app\n")
	}

	if _, err := os.Stat(filepath.Join(dest, ".github", "workflows", "ci.yml")); err != nil {
		t.Errorf(".github/workflows/ci.yml: expected still present: %v", err)
	}
	for _, rel := range []string{"GEMINI.md", ".cursor"} {
		if _, err := os.Stat(filepath.Join(dest, filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Errorf("expected %s to not exist, got err=%v", rel, err)
		}
	}
}

func TestAddAgents_CodexIsNoOp(t *testing.T) {
	dest := scaffoldDest(t)

	written, err := AddAgents(gonext.Templates, "templates", dest, "my-app", []string{"codex"}, false)
	if err != nil {
		t.Fatalf("AddAgents: unexpected error: %v", err)
	}
	if len(written) != 0 {
		t.Errorf("AddAgents written = %v, want empty", written)
	}

	for _, rel := range []string{"CLAUDE.md", "GEMINI.md", ".cursor", ".claude"} {
		if _, err := os.Stat(filepath.Join(dest, filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Errorf("expected %s to not exist, got err=%v", rel, err)
		}
	}
}

func TestAddAgents_RefusesExistingFiles(t *testing.T) {
	dest := scaffoldDest(t)
	if err := os.WriteFile(filepath.Join(dest, "GEMINI.md"), []byte("mine\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := AddAgents(gonext.Templates, "templates", dest, "my-app", []string{"claude", "gemini"}, false)
	if err == nil {
		t.Fatal("AddAgents: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "GEMINI.md") {
		t.Errorf("AddAgents error %q does not mention GEMINI.md", err)
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("AddAgents error %q does not mention --force", err)
	}

	gemini, readErr := os.ReadFile(filepath.Join(dest, "GEMINI.md"))
	if readErr != nil {
		t.Fatalf("reading GEMINI.md: %v", readErr)
	}
	if string(gemini) != "mine\n" {
		t.Errorf("GEMINI.md = %q, want unchanged %q", gemini, "mine\n")
	}
	if _, err := os.Stat(filepath.Join(dest, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Errorf("expected CLAUDE.md to not exist, got err=%v", err)
	}
}

func TestAddAgents_ForceOverwrites(t *testing.T) {
	dest := scaffoldDest(t)
	if err := os.WriteFile(filepath.Join(dest, "GEMINI.md"), []byte("mine\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	written, err := AddAgents(gonext.Templates, "templates", dest, "my-app", []string{"gemini"}, true)
	if err != nil {
		t.Fatalf("AddAgents: unexpected error: %v", err)
	}
	if !slices.Equal(written, []string{"GEMINI.md"}) {
		t.Errorf("AddAgents written = %v, want %v", written, []string{"GEMINI.md"})
	}

	gemini, err := os.ReadFile(filepath.Join(dest, "GEMINI.md"))
	if err != nil {
		t.Fatalf("reading GEMINI.md: %v", err)
	}
	if string(gemini) == "mine\n" {
		t.Errorf("GEMINI.md was not overwritten: %q", gemini)
	}
	if !strings.Contains(string(gemini), "AGENTS.md") {
		t.Errorf("GEMINI.md = %q, want it to mention AGENTS.md", gemini)
	}
}

// TestAddAgents_RefusesSymlinks pins that a symlink anywhere on an owned
// path is never written through, even with force.
func TestAddAgents_RefusesSymlinks(t *testing.T) {
	tests := []struct {
		name   string
		link   string // dest-relative path to plant as a symlink
		target string // outside-relative target; "" links the outside dir itself
		agent  string
	}{
		{name: "dangling leaf file", link: "GEMINI.md", target: "GEMINI.md", agent: "gemini"},
		{name: "owned directory", link: ".claude", target: "", agent: "claude"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dest := scaffoldDest(t)
			outside := t.TempDir()
			if err := os.Symlink(filepath.Join(outside, tt.target), filepath.Join(dest, tt.link)); err != nil {
				t.Fatalf("setup: %v", err)
			}

			_, err := AddAgents(gonext.Templates, "templates", dest, "my-app", []string{tt.agent}, true)
			if err == nil {
				t.Fatal("AddAgents: expected error for a symlinked destination, got nil")
			}
			if !strings.Contains(err.Error(), "symlink") || !strings.Contains(err.Error(), tt.link) {
				t.Errorf("AddAgents error = %q, want it to name the symlink %s", err, tt.link)
			}

			entries, err := os.ReadDir(outside)
			if err != nil {
				t.Fatalf("reading outside dir: %v", err)
			}
			if len(entries) != 0 {
				t.Errorf("wrote through the symlink: outside dir has %v", entries)
			}
		})
	}
}

func TestAddAgents_RejectsNonProject(t *testing.T) {
	dest := t.TempDir()

	_, err := AddAgents(gonext.Templates, "templates", dest, "my-app", []string{"claude"}, false)
	if err == nil {
		t.Fatal("AddAgents: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "AGENTS.md") {
		t.Errorf("AddAgents error %q does not mention AGENTS.md", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Errorf("expected CLAUDE.md to not exist, got err=%v", err)
	}
}
