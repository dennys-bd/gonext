package scaffold

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
)

func TestDecidePin(t *testing.T) {
	tests := map[string]struct {
		version   string
		settings  []debug.BuildSetting
		want      string
		published bool
	}{
		"published version, no vcs setting": {
			version:   "v0.3.0",
			want:      "v0.3.0",
			published: true,
		},
		"devel build": {
			version:   "(devel)",
			published: false,
		},
		"pseudo-version with vcs.revision (checkout build/install)": {
			version:   "v0.0.0-20260912225425-d0ae2746462c",
			settings:  []debug.BuildSetting{{Key: "vcs.revision", Value: "abc123"}},
			published: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			version, published := decidePin(tc.version, tc.settings)
			if published != tc.published {
				t.Errorf("published = %v, want %v", published, tc.published)
			}
			if version != tc.want {
				t.Errorf("version = %q, want %q", version, tc.want)
			}
		})
	}
}

func TestModuleEdit_PublishedVersion(t *testing.T) {
	defer stubReadBuildInfo(t, "v0.3.0", nil)()

	args, err := ModuleEdit(t.TempDir())
	if err != nil {
		t.Fatalf("ModuleEdit: %v", err)
	}
	want := "-require=" + ModulePath + "@v0.3.0"
	if len(args) != 1 || args[0] != want {
		t.Errorf("ModuleEdit args = %v, want [%s]", args, want)
	}
}

func TestModuleEdit_CheckoutBuild_OutsideCheckout(t *testing.T) {
	defer stubReadBuildInfo(t, "(devel)", nil)()

	args, err := ModuleEdit(t.TempDir())
	if err != nil {
		t.Fatalf("ModuleEdit: %v", err)
	}
	if len(args) != 1 || !strings.HasPrefix(args[0], "-replace="+ModulePath+"=") {
		t.Fatalf("ModuleEdit args = %v, want a -replace= argument", args)
	}
	path := strings.TrimPrefix(args[0], "-replace="+ModulePath+"=")
	if !filepath.IsAbs(path) {
		t.Errorf("replace path = %q, want an absolute path (dest is outside the checkout)", path)
	}
}

func TestModuleEdit_CheckoutBuild_InsideCheckout(t *testing.T) {
	defer stubReadBuildInfo(t, "(devel)", nil)()

	root, err := checkoutRoot()
	if err != nil {
		t.Fatalf("checkoutRoot: %v", err)
	}
	gomod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil || !strings.HasPrefix(string(gomod), "module "+ModulePath+"\n") {
		t.Fatalf("checkoutRoot = %s, which does not hold the %s go.mod (err: %v)", root, ModulePath, err)
	}
	dest := filepath.Join(root, "golden")

	args, err := ModuleEdit(dest)
	if err != nil {
		t.Fatalf("ModuleEdit: %v", err)
	}
	want := "-replace=" + ModulePath + "=.."
	if len(args) != 1 || args[0] != want {
		t.Errorf("ModuleEdit args = %v, want [%s]", args, want)
	}
}

func TestReplacePath(t *testing.T) {
	root := t.TempDir()

	tests := map[string]struct {
		dest string
		want string
	}{
		"dest inside root":        {dest: filepath.Join(root, "golden"), want: ".."},
		"dest nested inside root": {dest: filepath.Join(root, "golden", "sub"), want: "../.."},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := replacePath(tc.dest, root)
			if err != nil {
				t.Fatalf("replacePath: %v", err)
			}
			if got != tc.want {
				t.Errorf("replacePath(%q, %q) = %q, want %q", tc.dest, root, got, tc.want)
			}
		})
	}

	t.Run("dest outside root", func(t *testing.T) {
		other := t.TempDir()
		got, err := replacePath(other, root)
		if err != nil {
			t.Fatalf("replacePath: %v", err)
		}
		rootAbs, _ := filepath.Abs(root)
		if got != rootAbs {
			t.Errorf("replacePath(%q, %q) = %q, want %q", other, root, got, rootAbs)
		}
	})
}

// stubReadBuildInfo makes ModuleEdit see a binary built with the given main
// module version and build settings, restoring the real ReadBuildInfo after.
func stubReadBuildInfo(t *testing.T, version string, settings []debug.BuildSetting) func() {
	t.Helper()
	orig := ReadBuildInfo
	ReadBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: version}, Settings: settings}, true
	}
	return func() { ReadBuildInfo = orig }
}
