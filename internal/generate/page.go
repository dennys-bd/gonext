package generate

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"text/template"

	xexec "github.com/dennys-bd/gonext/internal/exec"
	"github.com/dennys-bd/gonext/internal/openapi"
)

//go:embed templates/*.tmpl
var pageTemplates embed.FS

var pageTmpl = template.Must(template.ParseFS(pageTemplates, "templates/*.tmpl"))

// formatFunc runs biome over the files Page just wrote; overridden in
// tests to avoid invoking pnpm for real.
var formatFunc = xexec.Run

// warnOut is where Page reports skipped properties and a formatting
// fallback; overridden in tests.
var warnOut io.Writer = os.Stderr

// pageTarget is one file Page writes, paired with the template that
// renders it.
type pageTarget struct {
	name     string // filename, within the route directory
	template string
}

// Page writes a Next.js route for operationID under frontend/app/<route> and
// returns the root-relative paths written. It never overwrites; a
// formatting failure is reported as a warning, not an error.
func Page(root, route, operationID string) ([]string, error) {
	doc, err := openapi.Load(root)
	if err != nil {
		return nil, err
	}
	op, err := doc.Operation(operationID)
	if err != nil {
		return nil, err
	}
	r, err := parseRoute(route)
	if err != nil {
		return nil, err
	}
	m, err := buildPageModel(doc, op, r)
	if err != nil {
		return nil, err
	}

	targets := pageTargets(m)
	dir := r.dir(root)
	rootRel := make([]string, len(targets))
	for i, t := range targets {
		rootRel[i] = path.Join("frontend", "app", r.raw, t.name)
	}
	if err := checkTargetsAbsent(dir, targets, rootRel); err != nil {
		return nil, err
	}

	rendered := make([][]byte, len(targets))
	for i, t := range targets {
		buf, err := renderTemplate(t.template, m)
		if err != nil {
			return nil, fmt.Errorf("rendering %s: %w", rootRel[i], err)
		}
		rendered[i] = buf
	}

	if err := os.MkdirAll(dir, dirMode); err != nil {
		return nil, fmt.Errorf("creating %s: %w", dir, err)
	}
	for i, t := range targets {
		if err := writeExclusive(filepath.Join(dir, t.name), rendered[i]); err != nil {
			return nil, err
		}
	}

	for _, w := range m.Warnings {
		fmt.Fprintln(warnOut, "warning:", w)
	}
	formatWritten(root, rootRel)

	return rootRel, nil
}

// pageTargets returns the files Page writes for m, in write order:
// GET operations get a page and its test; everything else
// additionally gets an action, a form and the action's test.
func pageTargets(m pageModel) []pageTarget {
	if m.IsGet {
		return []pageTarget{
			{"page.tsx", "get-page.tsx.tmpl"},
			{"page.test.tsx", "get-page.test.tsx.tmpl"},
		}
	}
	return []pageTarget{
		{"page.tsx", "mutation-page.tsx.tmpl"},
		{"actions.ts", "actions.ts.tmpl"},
		{m.FormFile, "form.tsx.tmpl"},
		{"actions.test.ts", "actions.test.ts.tmpl"},
		{"page.test.tsx", "mutation-page.test.tsx.tmpl"},
	}
}

// checkTargetsAbsent stats every target before anything is written,
// so a conflict on the last file still leaves the first untouched. A
// stat error other than "not exist" is reported rather than treated
// as absent.
func checkTargetsAbsent(dir string, targets []pageTarget, rootRel []string) error {
	for i, t := range targets {
		_, err := os.Stat(filepath.Join(dir, t.name))
		switch {
		case err == nil:
			return fmt.Errorf("%s already exists", rootRel[i])
		case errors.Is(err, fs.ErrNotExist):
			continue
		default:
			return fmt.Errorf("checking %s: %w", rootRel[i], err)
		}
	}
	return nil
}

func renderTemplate(name string, m pageModel) ([]byte, error) {
	var buf bytes.Buffer
	if err := pageTmpl.ExecuteTemplate(&buf, name, m); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeExclusive(dest string, data []byte) error {
	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fileMode)
	if err != nil {
		return fmt.Errorf("writing %s: %w", filepath.Base(dest), err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("writing %s: %w", filepath.Base(dest), err)
	}
	return f.Close()
}

// formatWritten runs formatFunc over rootRel (paths rooted at the
// project root) relative to frontend/, and reports a failure as a
// warning: the files are already on disk.
func formatWritten(root string, rootRel []string) {
	frontend := filepath.Join(root, "frontend")
	args := []string{"exec", "biome", "check", "--write"}
	for _, p := range rootRel {
		args = append(args, p[len("frontend/"):])
	}
	if err := formatFunc(context.Background(), frontend, "pnpm", args...); err != nil {
		fmt.Fprintf(warnOut, "warning: formatting failed (%v); run `pnpm format` in frontend/\n", err)
	}
}
