package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	gonext "github.com/dennys-bd/gonext"
	"github.com/dennys-bd/gonext/internal/scaffold"
)

// goldenSlug must stay in sync with cmd/golden's own goldenSlug; run `make
// golden` to regenerate golden/ if it ever changes.
const goldenSlug = "golden-app"

// goldenDir is the committed, runnable golden/ dev tree, resolved relative
// to this package's directory.
const goldenDir = "../../golden"

// TestCopy_GoldenSnapshot pins Copy's output for the real embedded
// templates/ tree byte-for-byte against the committed golden/ dev tree.
func TestCopy_GoldenSnapshot(t *testing.T) {
	dest := t.TempDir()

	if err := scaffold.Copy(gonext.Templates, "templates", dest, goldenSlug, nil); err != nil {
		t.Fatalf("Copy: unexpected error: %v", err)
	}

	assertTreesEqual(t, goldenDir, dest)
}

// generatedDirs are top-level dirs under golden/ that Copy() never writes,
// excluded from the comparison and never walked (pnpm's node_modules uses
// symlinks a naive file-read chokes on).
var generatedDirs = []string{
	"frontend/node_modules",
	"frontend/.next",
	"frontend/lib/api",
	"backend/bin",
}

// generatedFiles are top-level files under golden/ that a later runInit step
// produces rather than Copy().
var generatedFiles = []string{"go.mod", "go.sum", localConfig}

// toolStateDirs are gitignored agent-tooling state dirs that can appear at
// any depth (not just top-level), since a hook writes .omc/ relative to
// whatever directory a shell happens to be in.
var toolStateDirs = []string{".omc"}

// isGeneratedArtifact reports whether rel is a tool-generated path under
// golden/ that Copy never writes.
func isGeneratedArtifact(rel string) bool {
	if slices.Contains(generatedFiles, rel) {
		return true
	}
	for _, dir := range generatedDirs {
		if rel == dir || strings.HasPrefix(rel, dir+"/") {
			return true
		}
	}
	// Matched per path component so a directory merely sharing a
	// prefix (.omcfoo) is still compared.
	return slices.ContainsFunc(strings.Split(rel, "/"), func(part string) bool {
		return slices.Contains(toolStateDirs, part)
	})
}

func TestIsGeneratedArtifact(t *testing.T) {
	tests := []struct {
		rel  string
		want bool
	}{
		{rel: "go.mod", want: true},
		{rel: "go.sum", want: true},
		{rel: "mise.local.toml", want: true},
		{rel: "mise.local.toml.example", want: false},
		{rel: "frontend/node_modules", want: true},
		{rel: "frontend/node_modules/next/package.json", want: true},
		{rel: "frontend/.next/build-manifest.json", want: true},
		{rel: "frontend/lib/api/sdk.gen.ts", want: true},
		{rel: "frontend/lib/api-client.ts", want: false},
		{rel: "backend/bin/server", want: true},
		{rel: "backend/cmd/server/main.go", want: false},
		{rel: "frontend/package.json", want: false},
		{rel: "README.md", want: false},
		// Agent tool state, which lands at whatever depth a shell
		// happened to be running at rather than only the top level.
		{rel: ".omc", want: true},
		{rel: ".omc/state/idle-notif-cooldown.json", want: true},
		{rel: "backend/.omc/state/agent-replay.jsonl", want: true},
		{rel: "docs/bruno/users/.omc/state/throttle.json", want: true},
		// A name that merely starts with the same characters is not
		// tool state and must still be compared.
		{rel: "docs/.omcfoo/notes.md", want: false},
		{rel: "docs/omc/notes.md", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.rel, func(t *testing.T) {
			if got := isGeneratedArtifact(tt.rel); got != tt.want {
				t.Errorf("isGeneratedArtifact(%q) = %v, want %v", tt.rel, got, tt.want)
			}
		})
	}
}

// assertTreesEqual reports every path where the dest tree differs
// from the want tree: missing files, extra files, or differing
// content.
func assertTreesEqual(t *testing.T, want, dest string) {
	t.Helper()

	wantFiles := map[string][]byte{}
	err := filepath.WalkDir(want, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(want, path)
		if err != nil {
			return err
		}
		if d.IsDir() {
			if isGeneratedArtifact(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if isGeneratedArtifact(rel) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		wantFiles[rel] = data
		return nil
	})
	if err != nil {
		t.Fatalf("walking golden fixture %s: %v", want, err)
	}

	gotFiles := map[string][]byte{}
	err = filepath.WalkDir(dest, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dest, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		gotFiles[rel] = data
		return nil
	})
	if err != nil {
		t.Fatalf("walking Copy output %s: %v", dest, err)
	}

	for rel, wantData := range wantFiles {
		gotData, ok := gotFiles[rel]
		if !ok {
			t.Errorf("%s: missing from Copy output", rel)
			continue
		}
		if !bytes.Equal(gotData, wantData) {
			t.Errorf("%s: content differs from golden fixture", rel)
		}
	}
	for rel := range gotFiles {
		if _, ok := wantFiles[rel]; !ok {
			t.Errorf("%s: unexpected file in Copy output, not present in golden fixture", rel)
		}
	}
}
