package database

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// postgresUniqueViolationCode is Postgres' SQLSTATE for
// unique_violation.
const postgresUniqueViolationCode = "23505"

// RequireOneRow turns an update/delete that matched nothing into
// notFound, so a caller can tell "no such row" from "row left
// unchanged".
func RequireOneRow(res sql.Result, notFound error) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("reading affected rows: %w", err)
	}
	if affected == 0 {
		return notFound
	}
	return nil
}

// IsUniqueViolation reports whether err is Postgres' unique_violation (SQLSTATE 23505).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == postgresUniqueViolationCode
}
