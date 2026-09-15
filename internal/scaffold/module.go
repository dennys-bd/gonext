package scaffold

import (
	"fmt"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
)

// ReadBuildInfo is debug.ReadBuildInfo, indirected so tests can inject
// build info.
var ReadBuildInfo = debug.ReadBuildInfo

// ModuleEdit returns the `go mod edit` arguments that tie dest's gonext
// dependency to what this CLI was built from: -require of the published
// version, or -replace to the checkout when built from source.
func ModuleEdit(dest string) ([]string, error) {
	info, ok := ReadBuildInfo()
	if !ok {
		return nil, fmt.Errorf("reading build info: not available")
	}

	if version, published := decidePin(info.Main.Version, info.Settings); published {
		return []string{"-require=" + ModulePath + "@" + version}, nil
	}

	root, err := checkoutRoot()
	if err != nil {
		return nil, err
	}
	path, err := replacePath(dest, root)
	if err != nil {
		return nil, err
	}
	return []string{"-replace=" + ModulePath + "=" + path}, nil
}

// go run/go test stamp "(devel)"; go build/go install from a checkout stamp
// a pseudo-version plus vcs.revision. Neither names a published version.
func decidePin(version string, settings []debug.BuildSetting) (pinVersion string, published bool) {
	if version == "(devel)" {
		return "", false
	}
	for _, s := range settings {
		if s.Key == "vcs.revision" {
			return "", false
		}
	}
	return version, true
}

func checkoutRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok || !filepath.IsAbs(file) {
		return "", fmt.Errorf("resolving the gonext checkout: binary built with -trimpath; install a published version instead")
	}
	// file is <root>/internal/scaffold/module.go.
	return filepath.Dir(filepath.Dir(filepath.Dir(file))), nil
}

// Relative when dest lies inside root so a committed go.mod (golden/) stays
// portable across machines.
func replacePath(dest, root string) (string, error) {
	destAbs, err := filepath.Abs(dest)
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}

	if destAbs != rootAbs && !strings.HasPrefix(destAbs, rootAbs+string(filepath.Separator)) {
		return rootAbs, nil
	}
	rel, err := filepath.Rel(destAbs, rootAbs)
	if err != nil {
		return "", err
	}
	rel = filepath.ToSlash(rel)
	if !strings.HasPrefix(rel, ".") {
		rel = "./" + rel
	}
	return rel, nil
}
