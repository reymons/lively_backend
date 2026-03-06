package db

import (
	"context"
	"database/sql"

	"lively/core"
)

type DB interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)

	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row

	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Trx interface {
	DB
}

type Client interface {
	DB

	ExecTrx(ctx context.Context, fn func(trx Trx) error) error
}

// Maps SQL errors to core ones
func MapError(err error) error {
	if err == nil {
		return nil
	}
	if err == sql.ErrNoRows {
		return core.ErrEntityNotFound
	}
	return err
}
