package database

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestOfflineDB_NeverDials(t *testing.T) {
	start := time.Now()

	db := OfflineDB()
	if db == nil {
		t.Fatal("OfflineDB: expected non-nil *bun.DB")
	}

	if err := db.PingContext(context.Background()); !errors.Is(err, ErrOfflineDB) {
		t.Fatalf("PingContext: got %v, want an error satisfying errors.Is(err, ErrOfflineDB)", err)
	}

	var dest struct {
		ID string
	}
	err := db.NewSelect().Model(&dest).Scan(context.Background())
	if !errors.Is(err, ErrOfflineDB) {
		t.Fatalf("NewSelect().Scan: got %v, want an error satisfying errors.Is(err, ErrOfflineDB)", err)
	}

	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("OfflineDB took %v; expected no dial/backoff attempt", elapsed)
	}
}
