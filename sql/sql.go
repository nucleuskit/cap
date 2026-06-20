package sql

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidBatch = errors.New("invalid sql batch insert")

type Result interface {
	RowsAffected() (int64, error)
	LastInsertID() (int64, error)
}

type Dialect string

const (
	DialectMySQL    Dialect = "mysql"
	DialectPostgres Dialect = "postgres"
)

type Query struct {
	Statement string
	Args      []any
	Operation string
	Target    string
}

type QueryMetadata struct {
	Name          string
	Driver        string
	Operation     string
	Target        string
	Statement     string
	Args          []any
	BatchRows     int
	InTransaction bool
	RowsAffected  int64
	LastInsertID  int64
	StartedAt     time.Time
	Duration      time.Duration
	Err           error
}

type QueryHook interface {
	BeforeQuery(ctx context.Context, metadata QueryMetadata) context.Context
	AfterQuery(ctx context.Context, metadata QueryMetadata)
}

type QueryHookFuncs struct {
	Before func(ctx context.Context, metadata QueryMetadata) context.Context
	After  func(ctx context.Context, metadata QueryMetadata)
}

func (h QueryHookFuncs) BeforeQuery(ctx context.Context, metadata QueryMetadata) context.Context {
	if h.Before == nil {
		return ctx
	}
	return h.Before(ctx, metadata.Clone())
}

func (h QueryHookFuncs) AfterQuery(ctx context.Context, metadata QueryMetadata) {
	if h.After != nil {
		h.After(ctx, metadata.Clone())
	}
}

type Page struct {
	Number int
	Size   int
	Order  string
}

type BatchInsert struct {
	Table           string
	Columns         []string
	Rows            [][]any
	Dialect         Dialect
	MaxRows         int
	Ignore          bool
	OnConflict      []string
	OnDuplicateSets []string
}

type BatchStatement struct {
	Statement string
	Args      []any
	Rows      int
	Dialect   Dialect
}

type Row interface {
	Scan(dest ...any) error
}

type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Close() error
	Err() error
}

type Stmt interface {
	Exec(ctx context.Context, args ...any) (Result, error)
	Query(ctx context.Context, args ...any) (Rows, error)
	QueryRow(ctx context.Context, args ...any) Row
	Close() error
}

type Tx interface {
	Exec(ctx context.Context, query string, args ...any) (Result, error)
	Query(ctx context.Context, query string, args ...any) (Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) Row
	Prepare(ctx context.Context, query string) (Stmt, error)
	Commit() error
	Rollback() error
}

type DB interface {
	Exec(ctx context.Context, query string, args ...any) (Result, error)
	Query(ctx context.Context, query string, args ...any) (Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) Row
	Prepare(ctx context.Context, query string) (Stmt, error)
	Begin(ctx context.Context) (Tx, error)
	WithTransaction(ctx context.Context, fn func(context.Context, Tx) error) error
	BatchInsert(ctx context.Context, batch BatchInsert) (Result, error)
}

type txContextKey struct{}

func WithTx(ctx context.Context, tx Tx) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if tx == nil {
		return ctx
	}
	return context.WithValue(ctx, txContextKey{}, tx)
}

func TxFromContext(ctx context.Context) (Tx, bool) {
	if ctx == nil {
		return nil, false
	}
	tx, ok := ctx.Value(txContextKey{}).(Tx)
	return tx, ok
}

func Exec(ctx context.Context, db DB, query string, args ...any) (Result, error) {
	if tx, ok := TxFromContext(ctx); ok {
		return tx.Exec(ctx, query, args...)
	}
	return db.Exec(ctx, query, args...)
}

func QueryContext(ctx context.Context, db DB, query string, args ...any) (Rows, error) {
	if tx, ok := TxFromContext(ctx); ok {
		return tx.Query(ctx, query, args...)
	}
	return db.Query(ctx, query, args...)
}

func QueryRowContext(ctx context.Context, db DB, query string, args ...any) Row {
	if tx, ok := TxFromContext(ctx); ok {
		return tx.QueryRow(ctx, query, args...)
	}
	return db.QueryRow(ctx, query, args...)
}

func BuildBatchInsertStatements(batch BatchInsert) ([]BatchStatement, error) {
	if err := validateBatch(batch); err != nil {
		return nil, err
	}
	maxRows := batch.MaxRows
	if maxRows <= 0 || maxRows > len(batch.Rows) {
		maxRows = len(batch.Rows)
	}
	statements := make([]BatchStatement, 0, (len(batch.Rows)+maxRows-1)/maxRows)
	for start := 0; start < len(batch.Rows); start += maxRows {
		end := start + maxRows
		if end > len(batch.Rows) {
			end = len(batch.Rows)
		}
		statement, args := buildBatchInsertStatement(batch, batch.Rows[start:end])
		statements = append(statements, BatchStatement{
			Statement: statement,
			Args:      args,
			Rows:      end - start,
			Dialect:   batch.Dialect,
		})
	}
	return statements, nil
}

func MetadataForQuery(name string, driver string, statement string, args ...any) QueryMetadata {
	operation, target := ClassifyStatement(statement)
	return QueryMetadata{
		Name:      name,
		Driver:    driver,
		Operation: operation,
		Target:    target,
		Statement: statement,
		Args:      append([]any(nil), args...),
	}
}

func ClassifyStatement(statement string) (operation string, target string) {
	fields := strings.Fields(statement)
	if len(fields) == 0 {
		return "", ""
	}
	operation = strings.ToLower(fields[0])
	switch operation {
	case "insert":
		target = targetAfter(fields, "into")
	case "update":
		if len(fields) > 1 {
			target = trimIdentifier(fields[1])
		}
	case "delete":
		target = targetAfter(fields, "from")
	case "select":
		target = targetAfter(fields, "from")
	}
	return operation, target
}

func (m QueryMetadata) Clone() QueryMetadata {
	m.Args = append([]any(nil), m.Args...)
	return m
}

func validateBatch(batch BatchInsert) error {
	dialect := batch.Dialect
	if dialect == "" {
		dialect = DialectMySQL
	}
	if !validIdentifierPath(batch.Table) {
		return fmt.Errorf("%w: invalid table %q", ErrInvalidBatch, batch.Table)
	}
	if len(batch.Columns) == 0 {
		return fmt.Errorf("%w: columns are required", ErrInvalidBatch)
	}
	for _, column := range batch.Columns {
		if !validIdentifierPath(column) {
			return fmt.Errorf("%w: invalid column %q", ErrInvalidBatch, column)
		}
	}
	if len(batch.Rows) == 0 {
		return fmt.Errorf("%w: rows are required", ErrInvalidBatch)
	}
	for i, row := range batch.Rows {
		if len(row) != len(batch.Columns) {
			return fmt.Errorf("%w: row %d has %d values for %d columns", ErrInvalidBatch, i, len(row), len(batch.Columns))
		}
	}
	if batch.MaxRows < 0 {
		return fmt.Errorf("%w: max rows must be positive", ErrInvalidBatch)
	}
	for _, column := range batch.OnConflict {
		if !validIdentifierPath(column) {
			return fmt.Errorf("%w: invalid conflict column %q", ErrInvalidBatch, column)
		}
	}
	if dialect == DialectPostgres && len(batch.OnDuplicateSets) > 0 && len(batch.OnConflict) == 0 {
		return fmt.Errorf("%w: postgres upsert requires conflict columns", ErrInvalidBatch)
	}
	return nil
}

func buildBatchInsertStatement(batch BatchInsert, rows [][]any) (string, []any) {
	dialect := batch.Dialect
	if dialect == "" {
		dialect = DialectMySQL
	}
	builder := strings.Builder{}
	if batch.Ignore && dialect == DialectMySQL {
		builder.WriteString("INSERT IGNORE INTO ")
	} else {
		builder.WriteString("INSERT INTO ")
	}
	builder.WriteString(batch.Table)
	builder.WriteString(" (")
	builder.WriteString(strings.Join(batch.Columns, ", "))
	builder.WriteString(") VALUES ")
	args := make([]any, 0, len(rows)*len(batch.Columns))
	placeholder := 1
	for rowIndex, row := range rows {
		if rowIndex > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString("(")
		for columnIndex, value := range row {
			if columnIndex > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(batchPlaceholder(dialect, placeholder))
			placeholder++
			args = append(args, value)
		}
		builder.WriteString(")")
	}
	if batch.Ignore && dialect == DialectPostgres && len(batch.OnDuplicateSets) == 0 {
		builder.WriteString(" ON CONFLICT DO NOTHING")
	}
	if len(batch.OnDuplicateSets) > 0 {
		switch dialect {
		case DialectPostgres:
			if len(batch.OnConflict) > 0 {
				builder.WriteString(" ON CONFLICT (")
				builder.WriteString(strings.Join(batch.OnConflict, ", "))
				builder.WriteString(") DO UPDATE SET ")
				builder.WriteString(strings.Join(batch.OnDuplicateSets, ", "))
			}
		default:
			builder.WriteString(" ON DUPLICATE KEY UPDATE ")
			builder.WriteString(strings.Join(batch.OnDuplicateSets, ", "))
		}
	}
	return builder.String(), args
}

func batchPlaceholder(dialect Dialect, index int) string {
	if dialect == DialectPostgres {
		return "$" + strconv.Itoa(index)
	}
	return "?"
}

func targetAfter(fields []string, marker string) string {
	for i := 0; i < len(fields)-1; i++ {
		if strings.EqualFold(fields[i], marker) {
			return trimIdentifier(fields[i+1])
		}
	}
	return ""
}

func trimIdentifier(value string) string {
	return strings.Trim(value, " ,;()")
}

func validIdentifierPath(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	parts := strings.Split(value, ".")
	for _, part := range parts {
		if !validIdentifier(part) {
			return false
		}
	}
	return true
}

func validIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			continue
		}
		return false
	}
	return true
}
