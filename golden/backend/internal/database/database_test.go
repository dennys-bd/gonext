package database_test

import (
	"context"
	"testing"

	"golden-app/backend/internal/database"
	"golden-app/backend/internal/database/dbtest"
)

func TestConnect_Success(t *testing.T) {
	db, err := database.Connect(context.Background(), dbtest.DSN(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("ping after connect: %v", err)
	}
}

func TestConnect_InvalidDSN(t *testing.T) {
	_, err := database.Connect(context.Background(), "not-a-valid-dsn")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}
