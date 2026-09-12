// Package dbmigrate registers a generated project's per-domain
// migrations, orders them across domains by declared dependencies,
// and applies the pending ones with Bun's migration table for
// bookkeeping. It depends on Bun, which every generated backend
// already depends on, and on nothing from the CLI.
package dbmigrate

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/uptrace/bun"
)

// MigrationFunc is the shape of a migration's up and down functions,
// matching Bun's own migration signature.
type MigrationFunc func(ctx context.Context, db *bun.DB) error

// Migration identifies one registered migration.
type Migration struct {
	Domain  string
	Version string
	Name    string
}

// String is Domain + "/" + Version — the Bun row name and the key
// used in every error.
func (m Migration) String() string {
	return m.Domain + "/" + m.Version
}

// Dependency names another domain's migration that must run first.
type Dependency struct {
	Domain, Version string
}

// Option is what After returns; Register accepts any number.
type Option struct {
	dep Dependency
}

// After declares that this migration must run after version of
// domain (and so after every earlier version of that domain). Pass
// one per dependency; Register accepts any number.
func After(domain, version string) Option {
	return Option{dep: Dependency{Domain: domain, Version: version}}
}

type entry struct {
	m    Migration
	up   MigrationFunc
	down MigrationFunc
	deps []Dependency
}

// Registry is the set Register adds to. The package-level default is
// what Register and Apply use; tests build their own.
type Registry struct {
	entries map[string]entry
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{entries: make(map[string]entry)}
}

// Add records m with its up/down functions and cross-domain
// dependencies. A duplicate (Domain, Version) is an error rather than
// a panic, since Registry is also built directly by tests and
// Register's own callers.
func (r *Registry) Add(m Migration, up, down MigrationFunc, deps ...Dependency) error {
	key := m.String()
	if _, ok := r.entries[key]; ok {
		return fmt.Errorf("migration %s registered twice", m)
	}
	r.entries[key] = entry{m: m, up: up, down: down, deps: deps}
	return nil
}

// Order returns every registered migration in dependency order, ties
// broken by (Domain, Version) so runs are deterministic. Edges come
// from two sources: within a domain, each version depends on the
// previous registered version; across domains, each declared After
// target.
func (r *Registry) Order() ([]Migration, error) {
	// byDomain groups keys by domain, sorted by version, to build the
	// within-domain chain.
	byDomain := make(map[string][]string)
	for key, e := range r.entries {
		byDomain[e.m.Domain] = append(byDomain[e.m.Domain], key)
	}
	for domain := range byDomain {
		versions := byDomain[domain]
		sortStrings(versions)
		byDomain[domain] = versions
	}

	// dependsOn[key] is the set of keys key depends on (must come
	// after). indegree is len(dependsOn[key]).
	dependsOn := make(map[string][]string, len(r.entries))
	for key := range r.entries {
		dependsOn[key] = nil
	}
	for _, keys := range byDomain {
		for i := 1; i < len(keys); i++ {
			dependsOn[keys[i]] = append(dependsOn[keys[i]], keys[i-1])
		}
	}
	for key, e := range r.entries {
		for _, dep := range e.deps {
			target := Migration{Domain: dep.Domain, Version: dep.Version}.String()
			if _, ok := r.entries[target]; !ok {
				return nil, fmt.Errorf("migration %s depends on %s, which is not registered", key, target)
			}
			dependsOn[key] = append(dependsOn[key], target)
		}
	}

	// dependents is the reverse edge: target -> keys that depend on it.
	dependents := make(map[string][]string, len(r.entries))
	indegree := make(map[string]int, len(r.entries))
	for key, deps := range dependsOn {
		indegree[key] = len(deps)
		for _, dep := range deps {
			dependents[dep] = append(dependents[dep], key)
		}
	}

	order, err := kahn(r.entries, indegree, dependents)
	if err != nil {
		return nil, err
	}

	migrations := make([]Migration, len(order))
	for i, key := range order {
		migrations[i] = r.entries[key].m
	}
	return migrations, nil
}

// kahn runs Kahn's algorithm over the graph described by indegree and
// dependents, returning the ordered keys or a cycle error.
func kahn(entries map[string]entry, indegree map[string]int, dependents map[string][]string) ([]string, error) {
	remaining := make(map[string]int, len(indegree))
	for k, v := range indegree {
		remaining[k] = v
	}

	var ready []string
	for key, deg := range remaining {
		if deg == 0 {
			ready = append(ready, key)
		}
	}

	var order []string
	for len(ready) > 0 {
		// ponytail: linear min scan, n is dozens; container/heap if a
		// project ever has thousands.
		minIdx := 0
		for i := 1; i < len(ready); i++ {
			if lessKey(entries, ready[i], ready[minIdx]) {
				minIdx = i
			}
		}
		next := ready[minIdx]
		ready = append(ready[:minIdx], ready[minIdx+1:]...)
		order = append(order, next)
		delete(remaining, next)

		for _, dependent := range dependents[next] {
			remaining[dependent]--
			if remaining[dependent] == 0 {
				ready = append(ready, dependent)
			}
		}
	}

	if len(remaining) > 0 {
		return nil, cycleError(entries, remaining, dependents)
	}
	return order, nil
}

// lessKey orders two migration keys by (Domain, Version).
func lessKey(entries map[string]entry, a, b string) bool {
	ma, mb := entries[a].m, entries[b].m
	if ma.Domain != mb.Domain {
		return ma.Domain < mb.Domain
	}
	return ma.Version < mb.Version
}

// cycleError builds the "migration dependency cycle: ..." error for
// the nodes left with in-degree >= 1 after Kahn's algorithm stalls.
// Every remaining node depends on at least one other remaining node,
// since anything depending only on already-ordered nodes would have
// reached in-degree 0. Starting from the smallest remaining key and
// walking dependsOn (rebuilt here as the inverse of dependents,
// restricted to the remaining nodes) must therefore revisit a node,
// and that repeat bounds a cycle.
func cycleError(entries map[string]entry, remaining map[string]int, dependents map[string][]string) error {
	// dependsOn restricted to remaining nodes, derived from dependents.
	dependsOn := make(map[string][]string, len(remaining))
	for target, deps := range dependents {
		if _, ok := remaining[target]; !ok {
			continue
		}
		for _, dependent := range deps {
			if _, ok := remaining[dependent]; ok {
				dependsOn[dependent] = append(dependsOn[dependent], target)
			}
		}
	}

	var keys []string
	for key := range remaining {
		keys = append(keys, key)
	}
	sortStrings(keys)
	start := keys[0]

	var path []string
	seen := make(map[string]int)
	cur := start
	for {
		if idx, ok := seen[cur]; ok {
			path = append(path[idx:], cur)
			break
		}
		seen[cur] = len(path)
		path = append(path, cur)

		deps := dependsOn[cur]
		sortStrings(deps)
		cur = deps[0]
	}

	// Rotate path (which currently ends where it repeats) so it starts
	// at its smallest key. path's last element already equals its
	// first; drop the duplicate before rotating and re-close it.
	cycle := path[:len(path)-1]
	minIdx := 0
	for i := 1; i < len(cycle); i++ {
		if cycle[i] < cycle[minIdx] {
			minIdx = i
		}
	}
	rotated := append(append([]string(nil), cycle[minIdx:]...), cycle[:minIdx]...)
	rotated = append(rotated, rotated[0])

	return fmt.Errorf("migration dependency cycle: %s", strings.Join(rotated, " -> "))
}

// sortStrings sorts s in place; a tiny helper to avoid importing
// "sort" for a single call site each.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

var defaultRegistry = NewRegistry()

// Register records the calling file as a migration. It must be
// called from init() in a file at backend/<domain>/migrations/
// <NNNN>_<name>.go; any other path panics, since a migration in the
// wrong place is a programming error that should fail the build's
// first run, not be silently skipped.
func Register(up, down MigrationFunc, opts ...Option) {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		panic("dbmigrate: could not determine caller file")
	}
	m, err := parsePath(file)
	if err != nil {
		panic("dbmigrate: " + err.Error())
	}
	deps := make([]Dependency, len(opts))
	for i, opt := range opts {
		deps[i] = opt.dep
	}
	if err := defaultRegistry.Add(m, up, down, deps...); err != nil {
		panic("dbmigrate: " + err.Error())
	}
}

var pathRE = regexp.MustCompile(`backend/([a-z0-9_]+)/migrations/(\d{4})_([a-z0-9_]+)\.go$`)

// parsePath derives (domain, version, name) from a migration file's
// path, matched as a suffix so it holds under -trimpath and in any
// checkout location.
func parsePath(file string) (Migration, error) {
	match := pathRE.FindStringSubmatch(filepath.ToSlash(file))
	if match == nil {
		return Migration{}, fmt.Errorf("%s is not at backend/<domain>/migrations/<NNNN>_<name>.go", file)
	}
	return Migration{Domain: match[1], Version: match[2], Name: match[3]}, nil
}
