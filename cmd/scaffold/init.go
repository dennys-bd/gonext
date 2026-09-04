package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	gonext "github.com/dennys-bd/gonext"
	xexec "github.com/dennys-bd/gonext/internal/exec"
	"github.com/dennys-bd/gonext/internal/migrate"
	"github.com/dennys-bd/gonext/internal/scaffold"
)

// runInit implements `gonext init [name] [path]` and returns the
// process exit code.
func runInit(args []string) int {
	var name, path string
	if len(args) > 0 {
		name = args[0]
	}
	if len(args) > 1 {
		path = args[1]
	}

	slug, err := resolveSlug(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	dest, err := scaffold.ResolveDest(slug, path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	if err := scaffold.CheckEmpty(dest); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	if err := scaffold.Copy(gonext.Templates, "templates", dest, slug); err != nil {
		fmt.Fprintln(os.Stderr, "error: copying templates:", err)
		return 1
	}

	ctx := context.Background()

	if err := xexec.Run(ctx, dest, "go", "mod", "init", slug); err != nil {
		fmt.Fprintln(os.Stderr, "error: go mod init failed:", err)
		return 1
	}
	if err := pinGonextModule(dest); err != nil {
		fmt.Fprintln(os.Stderr, "error: pinning the gonext module failed:", err)
		return 1
	}
	if err := xexec.Run(ctx, dest, "go", "mod", "tidy"); err != nil {
		fmt.Fprintln(os.Stderr, "error: go mod tidy failed:", err)
		return 1
	}

	frontendDir := filepath.Join(dest, "frontend")
	if err := xexec.Run(ctx, frontendDir, "pnpm", "install"); err != nil {
		fmt.Fprintln(os.Stderr, "error: pnpm install failed:", err)
		return 1
	}

	if err := copyEnvFile(dest); err != nil {
		fmt.Fprintln(os.Stderr, "error: copying .env:", err)
		return 1
	}

	if err := bootstrapDatabase(ctx, dest); err != nil {
		fmt.Println("warning: database bootstrap failed:", err)
		fmt.Println("  run manually: make db-up && make migrate-up")
	}

	fmt.Println()
	fmt.Println("Created", dest)
	fmt.Println("Next steps:")
	fmt.Println("  cd", dest)

	return 0
}

// resolveSlug obtains and validates the project slug from name,
// prompting interactively when name is empty and stdin is a TTY.
func resolveSlug(name string) (string, error) {
	slug, err := scaffold.ResolveSlugArg(name, scaffold.IsTTY())
	switch {
	case err == nil:
		if valErr := scaffold.ValidateSlug(slug); valErr != nil {
			return "", valErr
		}
		return slug, nil
	case errors.Is(err, scaffold.ErrNameRequired):
		return "", err
	default:
		// errNeedsPrompt: fall through to the interactive prompt,
		// which re-prompts on invalid input via its own validator.
		return scaffold.PromptSlug()
	}
}

// copyEnvFile copies dest/.env.example to dest/.env verbatim.
func copyEnvFile(dest string) error {
	data, err := os.ReadFile(filepath.Join(dest, ".env.example"))
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dest, ".env"), data, 0o644)
}

// bootstrapDatabase brings up Postgres via Docker Compose and runs
// the generated project's own migrate binary. Any failure here is
// best-effort and reported to the caller as a warning, not a fatal
// error.
func bootstrapDatabase(ctx context.Context, dest string) error {
	if err := xexec.Run(ctx, dest, "docker", "compose", "up", "-d", "db"); err != nil {
		return fmt.Errorf("docker compose up: %w", err)
	}

	err := xexec.WaitHealthy(ctx, xexec.DefaultHealthTimeout, func(ctx context.Context) error {
		cmd := exec.CommandContext(ctx, "docker", "compose", "exec", "-T", "db", "pg_isready", "-U", "app", "-d", "app")
		cmd.Dir = dest
		return cmd.Run()
	})
	if err != nil {
		return fmt.Errorf("waiting for database: %w", err)
	}

	if err := migrate.Apply(ctx, dest); err != nil {
		return fmt.Errorf("running migration: %w", err)
	}
	return nil
}

// pinGonextModule records the exact gonext version this CLI was built
// against, before `go mod tidy` gets a chance to resolve something
// newer.
func pinGonextModule(dest string) error {
	require := scaffold.ModulePath + "@" + scaffold.ModuleVersion
	return xexec.Run(context.Background(), dest, "go", "mod", "edit", "-require="+require)
}
