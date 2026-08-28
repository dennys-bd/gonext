package scaffold

import (
	"bytes"
	"embed"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//go:embed all:testdata/fixture
var fixtureFS embed.FS

func TestCopy_SubstitutesTextFiles(t *testing.T) {
	dest := t.TempDir()

	if err := Copy(fixtureFS, "testdata/fixture", dest, "my-app"); err != nil {
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

	if err := Copy(fixtureFS, "testdata/fixture", dest, "my-app"); err != nil {
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

	if err := Copy(fixtureFS, "testdata/fixture", dest, "my-app"); err != nil {
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

	if err := Copy(fixtureFS, "testdata/fixture", dest, "my-app"); err != nil {
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
	// A real "go.mod" file can't live inside an embed.FS fixture here:
	// go:embed treats any directory containing one as a separate
	// module and either refuses to embed it (at the pattern root) or
	// silently drops it (in a subdirectory) - the same module-boundary
	// behavior that ruled out a nested go.mod under templates/backend
	// in the CLI itself. os.DirFS has no such restriction, so it's
	// used here instead to exercise the skip behavior directly.
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
	if err := Copy(os.DirFS(src), ".", dest, "my-app"); err != nil {
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
