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
	migrator, order, err := r.newMigrator(db)
	if err != nil {
		return nil, err
	}
	if err := initMigrator(ctx, migrator); err != nil {
		return nil, err
	}
	unlock, err := lockMigrator(ctx, migrator)
	if err != nil {
		return nil, err
	}
	defer unlock(&err)

	names, _, group, err := appliedNamesAndRows(ctx, migrator)
	if err != nil {
		return nil, err
	}

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

// newMigrator builds a Bun migrator over r's full registered order
// (both Apply and Migrate need the same one), returning that order
// alongside it so callers that need it (Apply) don't recompute it.
func (r *Registry) newMigrator(db *bun.DB) (*migrate.Migrator, []Migration, error) {
	order, err := r.Order()
	if err != nil {
		return nil, nil, err
	}
	set := migrate.NewMigrations()
	for _, m := range order {
		set.Add(migrate.Migration{Name: m.String(), Comment: m.Name})
	}
	return migrate.NewMigrator(db, set), order, nil
}

// initMigrator creates migrator's bookkeeping tables if they don't
// already exist.
func initMigrator(ctx context.Context, migrator *migrate.Migrator) error {
	if err := migrator.Init(ctx); err != nil {
		return fmt.Errorf("initializing migration tables: %w", err)
	}
	return nil
}

// lockMigrator takes migrator's run lock, wrapping a failure with a
// recovery hint since Bun's lock is a row with no expiry — a run
// killed mid-way leaves it behind. The returned unlock must be
// deferred by the caller with a pointer to its own named error
// return, so an unlock failure surfaces without masking an earlier one.
func lockMigrator(ctx context.Context, migrator *migrate.Migrator) (unlock func(*error), err error) {
	if err := migrator.Lock(ctx); err != nil {
		return nil, fmt.Errorf("%w (a killed run leaves its row in bun_migration_locks; delete it to recover)", err)
	}
	return func(errp *error) {
		if uerr := migrator.Unlock(ctx); uerr != nil && *errp == nil {
			*errp = fmt.Errorf("unlocking migrations: %w", uerr)
		}
	}, nil
}

// appliedNamesAndRows reads migrator's applied migrations once,
// returning the name set Plan needs, the rows (keyed by name)
// MarkUnapplied needs, and the next group id for a fresh MarkApplied.
func appliedNamesAndRows(ctx context.Context, migrator *migrate.Migrator) (names map[string]bool, rows map[string]migrate.Migration, nextGroup int64, err error) {
	done, err := migrator.AppliedMigrations(ctx)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("reading applied migrations: %w", err)
	}
	names = make(map[string]bool, len(done))
	rows = make(map[string]migrate.Migration, len(done))
	for _, d := range done {
		names[d.Name] = true
		rows[d.Name] = d
	}
	return names, rows, done.LastGroupID() + 1, nil
}
