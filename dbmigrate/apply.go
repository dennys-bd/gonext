package dbmigrate

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// Apply applies every registered migration not yet recorded in the
// database, in dependency order, and returns the ones it applied.
// It stops at the first failure: earlier ones stay applied and
// marked, the failing one is not marked, and the returned slice
// holds what was applied before it.
func Apply(ctx context.Context, db *bun.DB) ([]Migration, error) {
	return defaultRegistry.Apply(ctx, db)
}

// Apply is the Registry-scoped form of the package-level Apply; see
// its doc comment.
func (r *Registry) Apply(ctx context.Context, db *bun.DB) (applied []Migration, err error) {
	order, err := r.Order()
	if err != nil {
		return nil, err
	}

	set := migrate.NewMigrations()
	for _, m := range order {
		set.Add(migrate.Migration{Name: m.String(), Comment: m.Name})
	}
	migrator := migrate.NewMigrator(db, set)

	if err := migrator.Init(ctx); err != nil {
		return nil, fmt.Errorf("initializing migration tables: %w", err)
	}
	if err := migrator.Lock(ctx); err != nil {
		// Bun's lock is a row in bun_migration_locks with no expiry, so
		// a run killed mid-way leaves it behind; say how to recover.
		return nil, fmt.Errorf("%w (a killed run leaves its row in bun_migration_locks; delete it to recover)", err)
	}
	defer func() {
		if uerr := migrator.Unlock(ctx); uerr != nil && err == nil {
			err = fmt.Errorf("unlocking migrations: %w", uerr)
		}
	}()

	done, err := migrator.AppliedMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading applied migrations: %w", err)
	}
	names := make(map[string]bool, len(done))
	for _, d := range done {
		names[d.Name] = true
	}
	group := done.LastGroupID() + 1

	for _, m := range order {
		if names[m.String()] {
			continue
		}
		e := r.entries[m.String()]
		if err := e.up(ctx, db); err != nil {
			return applied, fmt.Errorf("applying %s %s: %w", m, m.Name, err)
		}
		if err := migrator.MarkApplied(ctx, &migrate.Migration{Name: m.String(), GroupID: group}); err != nil {
			return applied, fmt.Errorf("marking %s applied: %w", m, err)
		}
		applied = append(applied, m)
	}
	return applied, nil
}
