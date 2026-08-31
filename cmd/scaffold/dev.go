package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	gonext "github.com/dennys-bd/gonext"
	"github.com/dennys-bd/gonext/internal/dev"
)

// runDev implements `gonext dev` and returns the process exit code.
// It watches the current project's backend/ for .go changes,
// rebuilding and restarting the server until interrupted.
func runDev(args []string) int {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := dev.Run(ctx, gonext.Templates, cwd); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}
