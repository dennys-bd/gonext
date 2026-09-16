package openapi_test

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	"golden-app/backend/internal/config"
	"golden-app/backend/internal/openapi"
)

// operationIDs is every operation the document must carry.
var operationIDs = []string{
	"healthz", "readyz",
	"create-stub", "get-stub",
	"register-user", "login-user", "logout-user", "get-current-user",
	"confirm-user-email", "request-password-reset", "confirm-password-reset",
}

var errOffline = errors.New("test: database touched during registration")

// offlineConnector mirrors what `gonext openapi` hands Initialize: a
// *bun.DB that refuses to connect, so a stray query fails fast and named.
type offlineConnector struct{}

func (offlineConnector) Connect(context.Context) (driver.Conn, error) { return nil, errOffline }
func (offlineConnector) Driver() driver.Driver                        { return offlineDriver{} }

type offlineDriver struct{}

func (offlineDriver) Open(string) (driver.Conn, error) { return nil, errOffline }

func render(t *testing.T) []byte {
	t.Helper()
	cfg := config.Config{Env: "prod", Port: 8080, LogLevel: "info", LogFormat: "json", ShutdownTimeout: 10 * time.Second, RateLimitRPS: 20, RateLimitBurst: 40}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	db := bun.NewDB(sql.OpenDB(offlineConnector{}), pgdialect.New())

	spec, err := openapi.Initialize(cfg, logger, db)
	if err != nil {
		t.Fatalf("Initialize: unexpected error: %v", err)
	}
	doc, err := spec.API.OpenAPI().YAML()
	if err != nil {
		t.Fatalf("rendering document: %v", err)
	}
	return doc
}

func TestInitialize_RegistersEveryOperationWithoutInfrastructure(t *testing.T) {
	doc := string(render(t))
	for _, id := range operationIDs {
		if !strings.Contains(doc, "operationId: "+id+"\n") {
			t.Errorf("document missing operationId %q", id)
		}
	}
}

// The drift check depends on this: the same registrations yield the same
// bytes, whatever the environment says.
func TestInitialize_DocumentIsDeterministic(t *testing.T) {
	first := render(t)

	t.Setenv("ENV", "dev")
	t.Setenv("PORT", "1")
	t.Setenv("DATABASE_URL", "postgres://nope")
	second := render(t)

	if !bytes.Equal(first, second) {
		t.Errorf("document differs between renders / under a hostile environment")
	}
}
