package dbmigrate

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// Apply applies every registered migration not yet recorded in the
// database, in dependency order, and returns the ones it applied.
// It stops at the first failure, leaving earlier ones applied.
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

// newMigrator builds a Bun migrator over r's registered order.
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

// lockMigrator takes migrator's run lock. Bun's lock has no expiry,
// so a killed run leaves it behind; the wrapped error hints at that.
// Callers must defer the returned unlock with their own named error.
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

// appliedNamesAndRows reads migrator's applied migrations once: the
// name set, the rows keyed by name, and the next group id to use.
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
