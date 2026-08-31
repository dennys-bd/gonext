package database

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

type fakeResult struct {
	affected int64
	err      error
}

func (f fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (f fakeResult) RowsAffected() (int64, error)  { return f.affected, f.err }

func TestRequireOneRow_ReturnsNilWhenRowAffected(t *testing.T) {
	if err := RequireOneRow(fakeResult{affected: 1}, errors.New("not found")); err != nil {
		t.Errorf("RequireOneRow: unexpected error: %v", err)
	}
}

func TestRequireOneRow_ReturnsNotFoundWhenNoRowsAffected(t *testing.T) {
	notFound := errors.New("not found")
	err := RequireOneRow(fakeResult{affected: 0}, notFound)
	if !errors.Is(err, notFound) {
		t.Errorf("RequireOneRow: got %v, want %v", err, notFound)
	}
}

func TestRequireOneRow_WrapsRowsAffectedError(t *testing.T) {
	wantErr := errors.New("boom")
	err := RequireOneRow(fakeResult{err: wantErr}, errors.New("not found"))
	if !errors.Is(err, wantErr) {
		t.Errorf("RequireOneRow: expected wrapped %v, got %v", wantErr, err)
	}
}

func TestIsUniqueViolation_TrueForUniqueViolationCode(t *testing.T) {
	err := &pgconn.PgError{Code: "23505"}
	if !IsUniqueViolation(err) {
		t.Errorf("IsUniqueViolation: expected true for code 23505")
	}
}

func TestIsUniqueViolation_FalseForOtherCode(t *testing.T) {
	err := &pgconn.PgError{Code: "23503"}
	if IsUniqueViolation(err) {
		t.Errorf("IsUniqueViolation: expected false for code 23503")
	}
}

func TestIsUniqueViolation_FalseForNonPgError(t *testing.T) {
	if IsUniqueViolation(errors.New("plain error")) {
		t.Errorf("IsUniqueViolation: expected false for non-pgconn error")
	}
}
