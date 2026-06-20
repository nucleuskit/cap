package sql

import (
	"context"
	"errors"
)

var ErrNotConfigured = errors.New("sql not configured")

type noopDB struct{}

func NewNoop() *noopDB {
	return &noopDB{}
}

func (noopDB) Exec(context.Context, string, ...any) (Result, error) {
	return nil, ErrNotConfigured
}

func (noopDB) Query(context.Context, string, ...any) (Rows, error) {
	return nil, ErrNotConfigured
}

func (noopDB) QueryRow(context.Context, string, ...any) Row {
	return noopRow{}
}

func (noopDB) Prepare(context.Context, string) (Stmt, error) {
	return nil, ErrNotConfigured
}

func (noopDB) Begin(context.Context) (Tx, error) {
	return nil, ErrNotConfigured
}

func (noopDB) WithTransaction(context.Context, func(context.Context, Tx) error) error {
	return ErrNotConfigured
}

func (noopDB) BatchInsert(context.Context, BatchInsert) (Result, error) {
	return nil, ErrNotConfigured
}

type noopRow struct{}

func (noopRow) Scan(...any) error {
	return ErrNotConfigured
}
