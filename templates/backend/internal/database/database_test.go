package database

import (
	"context"
	"os"
	"testing"
)

func TestConnect_Success(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; run `make db-up` and export it to run this test")
	}

	db, err := Connect(context.Background(), dsn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("ping after connect: %v", err)
	}
}

func TestConnect_InvalidDSN(t *testing.T) {
	_, err := Connect(context.Background(), "not-a-valid-dsn")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}
