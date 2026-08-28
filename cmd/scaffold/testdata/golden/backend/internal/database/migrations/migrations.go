// Package migrations holds the backend's hand-written Postgres schema
// migrations, registered with Bun's migrator. Applied only via
// backend/cmd/migrate — never automatically on server boot.
package migrations

import "github.com/uptrace/bun/migrate"

// Migrations is the registered set of migrations backend/cmd/migrate
// applies against the database, in registration order.
var Migrations = migrate.NewMigrations()
