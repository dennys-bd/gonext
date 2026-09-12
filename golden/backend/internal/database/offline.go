package database

// This file exists solely so the OpenAPI document can be produced
// without infrastructure: OfflineDB hands InitializeSpec a *bun.DB
// that satisfies every domain's registration-time dependency without
// ever dialing a real database. Nothing calls this in a running
// server — see InitializeSpec in spec.go.

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

// ErrOfflineDB is returned by any attempt to actually use the DB
// OfflineDB hands back — connecting, pinging, or querying. Domain
// registration never does any of those (it only builds repositories
// that hold the handle), so this error should never surface outside
// this package's own tests.
var ErrOfflineDB = errors.New("database: offline DB used at runtime (spec-only; see InitializeSpec)")

// offlineConnector is a driver.Connector that refuses to connect,
// so building a *bun.DB over it never dials, never pings, and never
// retries.
type offlineConnector struct{}

func (offlineConnector) Connect(context.Context) (driver.Conn, error) {
	return nil, ErrOfflineDB
}

func (offlineConnector) Driver() driver.Driver {
	return offlineDriver{}
}

// offlineDriver backs offlineConnector.Driver. database/sql only
// calls Driver.Open on the legacy (non-Connector) path, which
// offlineConnector never takes, but sql.OpenDB requires a Driver()
// method, so this exists to keep the interface honest.
type offlineDriver struct{}

func (offlineDriver) Open(string) (driver.Conn, error) {
	return nil, ErrOfflineDB
}

// OfflineDB returns a *bun.DB that never dials a real database: every
// operation against it — Ping, a query, a transaction — returns
// ErrOfflineDB. It exists only to satisfy the *bun.DB dependency
// domain registration needs at wire-build time when producing the
// OpenAPI document offline.
func OfflineDB() *bun.DB {
	return bun.NewDB(sql.OpenDB(offlineConnector{}), pgdialect.New())
}
