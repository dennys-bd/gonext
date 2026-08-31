package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dennys-bd/gonext/internal/migrate"
)

// runMigrate implements `gonext migrate` and returns the process
// exit code. It applies the pending Postgres migrations for the
// generated project rooted at (or above) the current directory.
func runMigrate(args []string) int {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	root, err := migrate.ResolveRoot(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	if err := migrate.Apply(context.Background(), root); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}
