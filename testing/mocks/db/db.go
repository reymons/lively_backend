package mocks_db

import (
	"context"
	"database/sql"
	"errors"

	"lively/db"
)

var errUnimplemented = errors.New("unimplemented")

type dbClient struct{}

func NewClient() db.Client {
	return &dbClient{}
}

func (c *dbClient) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return nil, errUnimplemented
}

func (c *dbClient) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return nil
}

func (c *dbClient) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, errUnimplemented
}

func (c *dbClient) ExecTrx(ctx context.Context, fn func(trx db.Trx) error) error {
	return fn(c)
}
