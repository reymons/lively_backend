package db

import (
	"context"
	"database/sql"
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
