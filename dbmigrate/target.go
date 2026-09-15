package dbmigrate

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// Target names the state one domain should be brought to. Version
// "zero" means none of the domain's migrations applied.
type Target struct{ Domain, Version string }

// ParseTarget parses "<domain>/<version>" ("<domain>/zero" allowed).
func ParseTarget(s string) (Target, error) {
	domain, version, ok := strings.Cut(s, "/")
	if !ok || domain == "" || version == "" || strings.Contains(version, "/") {
		return Target{}, fmt.Errorf("target must be <domain>/<version>, got %q", s)
	}
	return Target{Domain: domain, Version: version}, nil
}

// Plan computes what bringing applied to target requires: rollback
// to unapply (reverse dependency order) and apply to run (dependency
// order). Both empty means target is already met.
func (r *Registry) Plan(target Target, applied map[string]bool) (rollback, apply []Migration, err error) {
	g, err := r.buildGraph()
	if err != nil {
		return nil, nil, err
	}

	domainKeys, ok := g.byDomain[target.Domain]
	if !ok {
		return nil, nil, fmt.Errorf("no such domain %s", target.Domain)
	}

	targetKey := ""
	if target.Version != "zero" {
		targetKey = (Migration{Domain: target.Domain, Version: target.Version}).String()
		if _, ok := r.entries[targetKey]; !ok {
			return nil, nil, fmt.Errorf("%s/%s is not a registered migration", target.Domain, target.Version)
		}
	}

	rollback, err = r.planRollback(g, domainKeys, target, applied)
	if err != nil {
		return nil, nil, err
	}
	apply, err = r.planApply(g, targetKey, applied)
	if err != nil {
		return nil, nil, err
	}
	return rollback, apply, nil
}

// planRollback finds every applied migration that must be unapplied
// to reach target, dependents ordered before their dependencies.
func (r *Registry) planRollback(g *graph, domainKeys []string, target Target, applied map[string]bool) ([]Migration, error) {
	set := make(map[string]bool)
	var queue []string
	for _, key := range domainKeys {
		if !applied[key] {
			continue
		}
		if target.Version == "zero" || r.entries[key].m.Version > target.Version {
			set[key] = true
			queue = append(queue, key)
		}
	}
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		for _, dependent := range g.dependents[key] {
			if !applied[dependent] || set[dependent] {
				continue
			}
			set[dependent] = true
			queue = append(queue, dependent)
		}
	}
	if len(set) == 0 {
		return nil, nil
	}

	// Kahn's algorithm again, but reversed: indegree counts dependents
	// here, so a migration is ready once nothing left still depends on it.
	indegree := make(map[string]int, len(set))
	reverseDeps := make(map[string][]string, len(set))
	for key := range set {
		indegree[key] = 0
	}
	for key := range set {
		for _, dependent := range g.dependents[key] {
			if set[dependent] {
				indegree[key]++
			}
		}
		for _, dep := range g.dependsOn[key] {
			if set[dep] {
				reverseDeps[key] = append(reverseDeps[key], dep)
			}
		}
	}

	order, err := kahn(r.entries, indegree, reverseDeps)
	if err != nil {
		return nil, err
	}
	migrations := make([]Migration, len(order))
	for i, key := range order {
		migrations[i] = r.entries[key].m
	}
	return migrations, nil
}

// planApply finds targetKey plus everything it transitively depends
// on, drops what's already applied, and orders the rest.
func (r *Registry) planApply(g *graph, targetKey string, applied map[string]bool) ([]Migration, error) {
	if targetKey == "" {
		return nil, nil
	}

	set := map[string]bool{targetKey: true}
	queue := []string{targetKey}
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		for _, dep := range g.dependsOn[key] {
			if set[dep] {
				continue
			}
			set[dep] = true
			queue = append(queue, dep)
		}
	}

	order, err := kahn(r.entries, g.indegree, g.dependents)
	if err != nil {
		return nil, err
	}
	var migrations []Migration
	for _, key := range order {
		if set[key] && !applied[key] {
			migrations = append(migrations, r.entries[key].m)
		}
	}
	return migrations, nil
}

// ErrNotConfirmed is what Migrate returns when confirm declines a
// non-empty rollback.
var ErrNotConfirmed = errors.New("rollback not confirmed")

// ErrStateChanged is returned by Migrate when the applied set changed
// between planning and locking, so the confirmed plan no longer
// describes what would run; rerun to plan against the new state.
var ErrStateChanged = errors.New("migration state changed since planning; rerun")

// Migrate brings target's domain to target.Version, rolling back
// before applying; confirm is asked before a non-empty rollback and
// must return true to proceed. It stops at the first failure.
func Migrate(ctx context.Context, db *bun.DB, target Target, confirm func(rollback []Migration) bool) ([]Migration, []Migration, error) {
	return defaultRegistry.Migrate(ctx, db, target, confirm)
}

// Migrate is the Registry-scoped form of the package-level Migrate;
// see its doc comment. It re-plans after locking and returns
// ErrStateChanged if a concurrent run changed the applied set.
func (r *Registry) Migrate(ctx context.Context, db *bun.DB, target Target, confirm func(rollback []Migration) bool) (rolledBack, applied []Migration, err error) {
	migrator, _, err := r.newMigrator(db)
	if err != nil {
		return nil, nil, err
	}
	if err := initMigrator(ctx, migrator); err != nil {
		return nil, nil, err
	}

	names, _, _, err := appliedNamesAndRows(ctx, migrator)
	if err != nil {
		return nil, nil, err
	}
	rollbackPlan, applyPlan, err := r.Plan(target, names)
	if err != nil {
		return nil, nil, err
	}
	if len(rollbackPlan) == 0 && len(applyPlan) == 0 {
		return nil, nil, nil
	}
	if len(rollbackPlan) > 0 && confirm != nil && !confirm(rollbackPlan) {
		return nil, nil, ErrNotConfirmed
	}

	unlock, err := lockMigrator(ctx, migrator)
	if err != nil {
		return nil, nil, err
	}
	defer unlock(&err)

	names, rows, group, err := appliedNamesAndRows(ctx, migrator)
	if err != nil {
		return nil, nil, err
	}
	freshRollback, freshApply, err := r.Plan(target, names)
	if err != nil {
		return nil, nil, err
	}
	if !slices.Equal(freshRollback, rollbackPlan) || !slices.Equal(freshApply, applyPlan) {
		return nil, nil, ErrStateChanged
	}

	for _, m := range rollbackPlan {
		// rows comes from the same read the plan was checked against; a
		// miss here is a bug, and a destructive loop must stop, not skip.
		row, ok := rows[m.String()]
		if !ok {
			return rolledBack, applied, fmt.Errorf("internal error: %s planned for rollback but not among applied rows", m)
		}
		e := r.entries[m.String()]
		if err := e.down(ctx, db); err != nil {
			return rolledBack, applied, fmt.Errorf("rolling back %s %s: %w", m, m.Name, err)
		}
		if err := migrator.MarkUnapplied(ctx, &row); err != nil {
			return rolledBack, applied, fmt.Errorf("marking %s unapplied: %w", m, err)
		}
		rolledBack = append(rolledBack, m)
	}

	for _, m := range applyPlan {
		e := r.entries[m.String()]
		if err := e.up(ctx, db); err != nil {
			return rolledBack, applied, fmt.Errorf("applying %s %s: %w", m, m.Name, err)
		}
		if err := migrator.MarkApplied(ctx, &migrate.Migration{Name: m.String(), GroupID: group}); err != nil {
			return rolledBack, applied, fmt.Errorf("marking %s applied: %w", m, err)
		}
		applied = append(applied, m)
	}

	return rolledBack, applied, nil
}
