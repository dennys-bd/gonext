package main

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"

	"[PROJECT-NAME]/backend/internal/config"
	"[PROJECT-NAME]/backend/internal/presentation/api"
	"[PROJECT-NAME]/backend/tracks"
	"[PROJECT-NAME]/backend/users"
)

// App bundles what main needs once wire has built and registered
// every dependency: the running Echo instance, the logger, and the
// shutdown timeout from config.
type App struct {
	Echo            *echo.Echo
	Logger          *slog.Logger
	Port            int
	ShutdownTimeout time.Duration
}

// NewApp assembles App. Its marker parameters (HealthzRegistered,
// ReadyzRegistered, tracks.Registered, users.Registered) aren't read;
// depending on them is what forces wire to run every registration
// before InitializeApp returns.
func NewApp(
	e *echo.Echo,
	logger *slog.Logger,
	cfg config.Config,
	_ api.HealthzRegistered,
	_ api.ReadyzRegistered,
	_ tracks.Registered,
	_ users.Registered,
) *App {
	return &App{
		Echo:            e,
		Logger:          logger,
		Port:            cfg.Port,
		ShutdownTimeout: cfg.ShutdownTimeout,
	}
}
