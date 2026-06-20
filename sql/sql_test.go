package sql

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestNoopDBImplementsSQLProtocol(t *testing.T) {
	db := NewNoop()
	var _ DB = db

	if _, err := db.Exec(context.Background(), "insert into audit(id) values (?)", "1"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Exec, got %v", err)
	}
	if _, err := db.Query(context.Background(), "select id from audit"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Query, got %v", err)
	}
	if err := db.QueryRow(context.Background(), "select id from audit").Scan(new(string)); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from QueryRow, got %v", err)
	}
}

func TestSQLOptions(t *testing.T) {
	hook := QueryHookFuncs{}
	options := NewOptions(WithDriver("postgres"), WithName("primary"), WithQueryHooks(hook))
	if options.Driver != "postgres" {
		t.Fatalf("expected driver postgres, got %q", options.Driver)
	}
	if options.Name != "primary" {
		t.Fatalf("expected name primary, got %q", options.Name)
	}
	if len(options.Hooks) != 1 {
		t.Fatalf("expected query hook")
	}
}

func TestBuildBatchInsertStatements(t *testing.T) {
	statements, err := BuildBatchInsertStatements(BatchInsert{
		Table:      "audit.logs",
		Columns:    []string{"id", "message"},
		Rows:       [][]any{{1, "created"}, {2, "updated"}},
		Dialect:    DialectPostgres,
		MaxRows:    1,
		OnConflict: []string{"id"},
		OnDuplicateSets: []string{
			"message = EXCLUDED.message",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(statements) != 2 {
		t.Fatalf("expected split statements, got %d", len(statements))
	}
	if statements[0].Statement != "INSERT INTO audit.logs (id, message) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET message = EXCLUDED.message" {
		t.Fatalf("unexpected statement: %s", statements[0].Statement)
	}
	if !reflect.DeepEqual(statements[0].Args, []any{1, "created"}) {
		t.Fatalf("unexpected args: %#v", statements[0].Args)
	}
	if statements[1].Statement != "INSERT INTO audit.logs (id, message) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET message = EXCLUDED.message" {
		t.Fatalf("unexpected second statement: %s", statements[1].Statement)
	}
}

func TestTxContextHelpers(t *testing.T) {
	tx := fakeTx{}
	ctx := WithTx(context.Background(), tx)
	got, ok := TxFromContext(ctx)
	if !ok {
		t.Fatal("expected tx from context")
	}
	if got != tx {
		t.Fatal("unexpected tx")
	}
}

func TestQueryMetadataClassifiesStatements(t *testing.T) {
	metadata := MetadataForQuery("primary", "database/sql", "SELECT * FROM accounts WHERE id = ?", 1)
	if metadata.Operation != "select" || metadata.Target != "accounts" {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
	metadata.Args[0] = 2
	clone := metadata.Clone()
	if clone.Args[0] != 2 {
		t.Fatalf("expected cloned args")
	}
}

type fakeTx struct{}

func (fakeTx) Exec(context.Context, string, ...any) (Result, error) { return nil, nil }
func (fakeTx) Query(context.Context, string, ...any) (Rows, error)  { return nil, nil }
func (fakeTx) QueryRow(context.Context, string, ...any) Row         { return noopRow{} }
func (fakeTx) Prepare(context.Context, string) (Stmt, error)        { return nil, nil }
func (fakeTx) Commit() error                                        { return nil }
func (fakeTx) Rollback() error                                      { return nil }
