package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	gonext "github.com/dennys-bd/gonext"
	"github.com/dennys-bd/gonext/internal/scaffold"
)

// goldenSlug is the project slug used to generate the golden fixture
// tree. It must stay in sync with the slug baked into
// testdata/golden's substituted content: regenerate the fixture (see
// TestCopy_GoldenSnapshot) if it ever changes.
const goldenSlug = "golden-app"

const goldenDir = "testdata/golden"

// TestCopy_GoldenSnapshot pins the real, full templates/ tree's
// copy+substitution output byte-for-byte against a checked-in golden
// fixture. Unlike internal/scaffold's own copy tests, which exercise
// Copy against a small synthetic fixture, this test runs Copy against
// the actual embedded templates/ tree used by `gonext init`, so it
// catches unintended output changes from either the copy/substitution
// logic or edits to templates/ itself. It hits no network or Docker,
// so it runs in the normal fast `go test` loop.
//
// When a template change is intentional, regenerate the fixture with:
//
//	UPDATE_GOLDEN=1 go test ./cmd/scaffold/ -run TestCopy_GoldenSnapshot
//
// then review the resulting testdata/golden diff before committing it.
func TestCopy_GoldenSnapshot(t *testing.T) {
	dest := t.TempDir()

	if err := scaffold.Copy(gonext.Templates, "templates", dest, goldenSlug); err != nil {
		t.Fatalf("Copy: unexpected error: %v", err)
	}

	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := updateGolden(dest); err != nil {
			t.Fatalf("updating golden fixture: %v", err)
		}
		t.Logf("regenerated %s from actual Copy output; review the diff before committing", goldenDir)
		return
	}

	assertTreesEqual(t, goldenDir, dest)
}

// assertTreesEqual reports every path where the dest tree differs
// from the want tree: missing files, extra files, or differing
// content.
func assertTreesEqual(t *testing.T, want, dest string) {
	t.Helper()

	wantFiles := map[string][]byte{}
	err := filepath.WalkDir(want, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(want, path)
		if err != nil {
			return err
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

// updateGolden replaces testdata/golden with a copy of dest.
func updateGolden(dest string) error {
	if err := os.RemoveAll(goldenDir); err != nil {
		return err
	}
	return filepath.WalkDir(dest, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dest, path)
		if err != nil {
			return err
		}
		target := filepath.Join(goldenDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}
