package generate

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dennys-bd/gonext/internal/openapi"
)

// Route is one contract operation and the frontend/app route
// directories whose client code calls it.
type Route struct {
	OperationID string
	Method      string
	Path        string
	CallName    string
	UsedBy      []string
}

// appSource is one .ts/.tsx file under frontend/app: its directory
// relative to frontend/, slash-separated, and its contents.
type appSource struct {
	dir     string
	content string
}

// Routes returns one Route per operation in docs/openapi.yaml under
// root, in Document.Operations' order. A missing frontend/app is not
// an error: every Route's UsedBy is empty.
func Routes(root string) ([]Route, error) {
	doc, err := openapi.Load(root)
	if err != nil {
		return nil, err
	}
	sources, err := appSources(filepath.Join(root, "frontend", "app"))
	if err != nil {
		return nil, err
	}

	ops := doc.Operations()
	routes := make([]Route, len(ops))
	for i, op := range ops {
		call := callName(&op)
		routes[i] = Route{
			OperationID: op.OperationID,
			Method:      op.Method,
			Path:        op.Path,
			CallName:    call,
			UsedBy:      usedBy(sources, call),
		}
	}
	return routes, nil
}

func appSources(appDir string) ([]appSource, error) {
	if _, err := os.Stat(appDir); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", appDir, err)
	}

	frontend := filepath.Dir(appDir)
	var sources []appSource
	err := filepath.WalkDir(appDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("reading %s: %w", p, err)
		}
		if d.IsDir() {
			if d.Name() == "node_modules" || d.Name() == ".next" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".ts") && !strings.HasSuffix(p, ".tsx") {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("reading %s: %w", p, err)
		}
		rel, err := filepath.Rel(frontend, filepath.Dir(p))
		if err != nil {
			return err
		}
		sources = append(sources, appSource{dir: filepath.ToSlash(rel), content: string(data)})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return sources, nil
}

func usedBy(sources []appSource, call string) []string {
	set := make(map[string]bool)
	for _, s := range sources {
		// ponytail: substring match on the bare call name; a longer call
		// sharing this prefix (api.users.getCurrentUserSettings) would
		// also match here. Tighten to call+"(" if that ever bites.
		if strings.Contains(s.content, call) {
			set[s.dir] = true
		}
	}
	dirs := make([]string, 0, len(set))
	for d := range set {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	return dirs
}
