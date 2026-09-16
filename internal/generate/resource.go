package generate

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"go/format"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"text/template"

	"github.com/dennys-bd/gonext/internal/project"
)

//go:embed templates/resource/*.tmpl
var resourceTemplates embed.FS

var resourceGoTmpl = template.Must(template.ParseFS(resourceTemplates, "templates/resource/*.go.tmpl"))

// resourceFile is one rendered file, keyed by its root-relative slash path.
type resourceFile struct {
	rel  string
	data []byte
}

type goTarget struct {
	rel, template string
}

// goTargets lists the Go files a resource writes under backend/<domain>,
// in the spec's listing order, with the migration last at migrationRel.
func goTargets(m resourceModel, migrationRel string) []goTarget {
	base := path.Join("backend", m.Domain)
	return []goTarget{
		{path.Join(base, "domain", m.Name+".go"), "domain.go.tmpl"},
		{path.Join(base, "internal/application", m.Name+"_service.go"), "service.go.tmpl"},
		{path.Join(base, "internal/application", m.Name+"_service_test.go"), "service_test.go.tmpl"},
		{path.Join(base, "internal/infrastructure/memory", m.Name+"_repository.go"), "memory.go.tmpl"},
		{path.Join(base, "internal/infrastructure/memory", m.Name+"_repository_test.go"), "memory_test.go.tmpl"},
		{path.Join(base, "internal/infrastructure/postgres", m.Name+"_repository.go"), "postgres.go.tmpl"},
		{path.Join(base, "internal/infrastructure/postgres", m.Name+"_repository_test.go"), "postgres_test.go.tmpl"},
		{path.Join(base, "internal/presentation", m.Name+".go"), "presentation.go.tmpl"},
		{path.Join(base, "internal/presentation", m.Name+"_test.go"), "presentation_test.go.tmpl"},
		{migrationRel, "migration.go.tmpl"},
	}
}

// renderResource renders every Go file for m, gofmt-formatted, in
// goTargets order. A template slip surfaces here as an error rather
// than as an unparsable file on disk.
func renderResource(m resourceModel, migrationRel string) ([]resourceFile, error) {
	targets := goTargets(m, migrationRel)
	files := make([]resourceFile, 0, len(targets))
	for _, t := range targets {
		var buf bytes.Buffer
		if err := resourceGoTmpl.ExecuteTemplate(&buf, t.template, m); err != nil {
			return nil, fmt.Errorf("rendering %s: %w", t.rel, err)
		}
		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			return nil, fmt.Errorf("formatting %s: %w", t.rel, err)
		}
		files = append(files, resourceFile{rel: t.rel, data: formatted})
	}
	return files, nil
}

// ResourceResult is what Resource wrote: every created file, the facade
// it edited, the Bruno requests whose seq it rewrote and the migration
// the developer applies next, all as root-relative slash paths.
type ResourceResult struct {
	Created    []string
	Facade     string
	Renumbered []string
	Migration  string
}

// Resource writes a CRUD slice for spec into backend/<domain> and
// docs/bruno/<domain> and wires it into the domain facade. It never
// overwrites a generated target; the caller refreshes the OpenAPI
// contract afterwards.
func Resource(root string, spec ResourceSpec) (ResourceResult, error) {
	if err := checkDomain(root, spec.Domain); err != nil {
		return ResourceResult{}, err
	}
	module, err := project.ModulePath(root)
	if err != nil {
		return ResourceResult{}, err
	}
	m, err := buildResourceModel(module, spec)
	if err != nil {
		return ResourceResult{}, err
	}

	facadeRel := path.Join("backend", m.Domain, m.Domain+".go")
	facadePath := filepath.Join(root, filepath.FromSlash(facadeRel))
	src, err := os.ReadFile(facadePath)
	if err != nil {
		reason := err.Error()
		if errors.Is(err, fs.ErrNotExist) {
			reason = "no such file"
		}
		return ResourceResult{}, wireError(facadeRel, facadeInfo{api: "api", db: "db", logger: "logger"}, m, reason)
	}
	info, err := inspectFacade(facadeRel, src, m)
	if err != nil {
		return ResourceResult{}, err
	}

	brunoDir := filepath.Join(root, "docs", "bruno", m.Domain)
	st, err := brunoFolder(brunoDir, m.DomainPascal)
	if err != nil {
		return ResourceResult{}, err
	}
	m.NewPackages = newPackages(root, m)
	version, err := nextVersion(filepath.Join(root, "backend", m.Domain, "migrations"))
	if err != nil {
		return ResourceResult{}, err
	}
	migrationRel := path.Join("backend", m.Domain, "migrations", version+"_create_"+m.Plural+".go")
	plan := planBruno(m, st)

	var rels []string
	for _, t := range goTargets(m, migrationRel) {
		rels = append(rels, t.rel)
	}
	for _, t := range plan.targets {
		rels = append(rels, path.Join("docs", "bruno", m.Domain, t.name+".bru"))
	}
	if err := checkTargetsAbsent(root, rels); err != nil {
		return ResourceResult{}, err
	}

	goFiles, err := renderResource(m, migrationRel)
	if err != nil {
		return ResourceResult{}, err
	}
	bruFiles, err := renderBruno(m, plan.targets)
	if err != nil {
		return ResourceResult{}, err
	}
	wired, err := wireFacade(src, info, m)
	if err != nil {
		return ResourceResult{}, err
	}

	// Write phase: nothing above touched the disk.
	result := ResourceResult{Facade: facadeRel}
	for _, f := range append(goFiles, bruFiles...) {
		if f.rel == migrationRel {
			// writeMigration recomputes the number; it agrees with the
			// pre-flight unless a migration raced in, in which case
			// O_EXCL on the pre-flight name would have failed anyway.
			rel, err := writeMigration(root, m.Domain, "create_"+m.Plural, string(f.data))
			if err != nil {
				return result, err
			}
			result.Migration = rel
			result.Created = append(result.Created, rel)
			continue
		}
		dest := filepath.Join(root, filepath.FromSlash(f.rel))
		if err := os.MkdirAll(filepath.Dir(dest), dirMode); err != nil {
			return result, fmt.Errorf("creating %s: %w", filepath.Dir(dest), err)
		}
		if err := writeExclusive(dest, f.data); err != nil {
			return result, err
		}
		result.Created = append(result.Created, f.rel)
	}
	for _, name := range slices.Sorted(maps.Keys(plan.renumber)) {
		if err := renumberSeq(filepath.Join(brunoDir, name), plan.renumber[name]); err != nil {
			return result, err
		}
		result.Renumbered = append(result.Renumbered, path.Join("docs", "bruno", m.Domain, name))
	}
	if err := os.WriteFile(facadePath, wired, fileMode); err != nil {
		return result, fmt.Errorf("writing %s: %w", facadeRel, err)
	}
	return result, nil
}

// newPackages reports which of the resource's package directories this
// run creates, so only those get a package doc comment.
func newPackages(root string, m resourceModel) map[string]bool {
	base := filepath.Join(root, "backend", m.Domain)
	dirs := map[string]string{
		"domain":       "domain",
		"application":  "internal/application",
		"memory":       "internal/infrastructure/memory",
		"postgres":     "internal/infrastructure/postgres",
		"presentation": "internal/presentation",
	}
	out := map[string]bool{}
	for key, dir := range dirs {
		if _, err := os.Stat(filepath.Join(base, filepath.FromSlash(dir))); err != nil {
			out[key] = true
		}
	}
	return out
}
