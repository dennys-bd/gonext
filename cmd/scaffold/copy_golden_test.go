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

// goldenSlug is the project slug used to generate the golden fixture
// tree. It must stay in sync with cmd/golden's own goldenSlug and the
// slug baked into golden/'s substituted content: run `make golden` to
// regenerate golden/ if it ever changes.
const goldenSlug = "golden-app"

// goldenDir is the committed, runnable golden/ dev tree at the repo
// root (see docs/superpowers/specs/2026-08-28-golden-app-runnable-tree-design.md),
// resolved relative to this package's directory.
const goldenDir = "../../golden"

// TestCopy_GoldenSnapshot pins the real, full templates/ tree's
// copy+substitution output byte-for-byte against the committed
// golden/ dev tree. Unlike internal/scaffold's own copy tests, which
// exercise Copy against a small synthetic fixture, this test runs
// Copy against the actual embedded templates/ tree used by `gonext
// init`, so it catches unintended output changes from either the
// copy/substitution logic or edits to templates/ itself. It hits no
// network or Docker, so it runs in the normal fast `go test` loop.
//
// golden/'s own go.mod/go.sum and its tool-generated build output
// (frontend/node_modules/, frontend/.next/, backend/bin/) are
// excluded from the comparison via isGeneratedArtifact: Copy never
// writes those, so they were never part of the comparison even when
// this test's fixture was testdata/golden (which never had them).
// Agent tool state (.omc/) is excluded the same way, at any depth —
// see toolStateDirs.
//
// When a template change is intentional, regenerate golden/ with:
//
//	make golden
//
// then review the resulting golden/ diff before committing it.
func TestCopy_GoldenSnapshot(t *testing.T) {
	dest := t.TempDir()

	if err := scaffold.Copy(gonext.Templates, "templates", dest, goldenSlug); err != nil {
		t.Fatalf("Copy: unexpected error: %v", err)
	}

	assertTreesEqual(t, goldenDir, dest)
}

// generatedDirs are top-level directories under golden/ that Copy()
// never writes: dependencies pnpm/go install and build output `make
// golden`'s underlying `gonext init` run produces. They're excluded
// from the golden-snapshot comparison — and, critically, never
// descended into during the walk below, since pnpm's node_modules
// layout uses symlinked directories that a naive file-read chokes on.
var generatedDirs = []string{
	"frontend/node_modules",
	"frontend/.next",
	"backend/bin",
}

// generatedFiles are top-level files under golden/ that Copy() never
// writes because a later runInit step produces them: go.mod/go.sum
// (via `go mod init`/`tidy`) and .env (copied from the .env.example
// Copy() does write).
var generatedFiles = []string{"go.mod", "go.sum", ".env"}

// toolStateDirs are directories agent tooling writes its own runtime
// state into. Unlike generatedDirs these are not anchored to the top
// level: a hook writes .omc/ relative to whatever directory a shell
// happens to be sitting in, so one can appear at any depth under
// golden/ simply because someone ran a command there. They are
// gitignored, but this test walks the working tree rather than the
// index, so without excluding them a stray hook write fails the
// comparison as an unexpected file.
var toolStateDirs = []string{".omc"}

// isGeneratedArtifact reports whether rel is a tool-generated path
// under golden/ that Copy never writes and so must be excluded from
// the golden-snapshot comparison.
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
		{rel: ".env", want: true},
		{rel: ".env.example", want: false},
		{rel: "frontend/node_modules", want: true},
		{rel: "frontend/node_modules/next/package.json", want: true},
		{rel: "frontend/.next/build-manifest.json", want: true},
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
