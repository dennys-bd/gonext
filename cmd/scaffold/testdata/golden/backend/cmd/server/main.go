// Package main runs the backend HTTP server: InitializeApp (see
// wire.go/wire_gen.go) builds the config, domains, and DB connection,
// then main just starts the server and shuts down gracefully on
// SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	app, cleanup, err := InitializeApp(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "initializing app: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr := fmt.Sprintf(":%d", app.Port)
	go func() {
		if err := app.Echo.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.Logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-stopCtx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), app.ShutdownTimeout)
	defer cancel()

	if err := app.Echo.Shutdown(shutdownCtx); err != nil {
		app.Logger.Error("shutdown error", "err", err)
		cancel()   // cancel is idempotent; run explicitly since os.Exit skips deferred calls
		os.Exit(1) //nolint:gocritic // cancel() called explicitly above
	}
}
