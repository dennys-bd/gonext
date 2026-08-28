// Command migrate applies and scaffolds the backend's Postgres
// migrations, via Bun's migrate.Migrator. It is never run
// automatically by the server — schema changes are a deliberate,
// explicit step.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"

	"personal-life/backend/internal/config"
	"personal-life/backend/internal/database"
	"personal-life/backend/internal/database/migrations"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: migrate <up|create> [args...]")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connecting to database: %v\n", err)
		os.Exit(1)
	}

	code := runCommand(ctx, db)
	_ = db.Close()
	os.Exit(code)
}

func runCommand(ctx context.Context, db *bun.DB) int {
	migrator := migrate.NewMigrator(db, migrations.Migrations)

	switch os.Args[1] {
	case "up":
		if err := runUp(ctx, migrator); err != nil {
			fmt.Fprintf(os.Stderr, "migrate up: %v\n", err)
			return 1
		}
	case "create":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: migrate create <name>")
			return 1
		}
		if err := runCreate(ctx, migrator, os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "migrate create: %v\n", err)
			return 1
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q; usage: migrate <up|create> [args...]\n", os.Args[1])
		return 1
	}
	return 0
}

func runUp(ctx context.Context, migrator *migrate.Migrator) error {
	if err := migrator.Init(ctx); err != nil {
		return fmt.Errorf("initializing migration tables: %w", err)
	}
	group, err := migrator.Migrate(ctx)
	if err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}
	if group.IsZero() {
		fmt.Println("no new migrations to run")
		return nil
	}
	fmt.Printf("migrated to %s\n", group)
	return nil
}

func runCreate(ctx context.Context, migrator *migrate.Migrator, name string) error {
	if err := migrator.Init(ctx); err != nil {
		return fmt.Errorf("initializing migration tables: %w", err)
	}
	mf, err := migrator.CreateGoMigration(ctx, name)
	if err != nil {
		return fmt.Errorf("creating migration file: %w", err)
	}
	fmt.Printf("created migration %s\n", mf.Path)
	return nil
}
