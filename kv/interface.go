package kv

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrNotFound         = errors.New("kv key not found")
	ErrCASConflict      = errors.New("kv cas conflict")
	ErrBatchFailed      = errors.New("kv batch failed")
	ErrInvalidOperation = errors.New("kv invalid operation")
)

type OperationKind string

const (
	OperationGet    OperationKind = "get"
	OperationPut    OperationKind = "put"
	OperationDelete OperationKind = "delete"
)

type Entry struct {
	Key       string
	Value     []byte
	Version   uint64
	ExpiresAt int64
}

type Operation struct {
	Kind    OperationKind
	Key     string
	Value   []byte
	Options WriteOptions
}

type Result struct {
	Operation Operation
	Entry     Entry
	Deleted   bool
	Err       error
}

type Store interface {
	Get(ctx context.Context, key string) (Entry, error)
	Put(ctx context.Context, key string, value []byte, options ...WriteOption) (Entry, error)
	Delete(ctx context.Context, key string, options ...WriteOption) error
	Batch(ctx context.Context, operations ...Operation) ([]Result, error)
}

type CASConflictError struct {
	Key      string
	Expected uint64
	Actual   uint64
}

func (e CASConflictError) Error() string {
	return fmt.Sprintf("%s: key %q expected version %d got %d", ErrCASConflict, e.Key, e.Expected, e.Actual)
}

func (e CASConflictError) Is(target error) bool {
	return target == ErrCASConflict
}

type BatchError struct {
	Results []Result
}

func (e BatchError) Error() string {
	for _, result := range e.Results {
		if result.Err != nil {
			return fmt.Sprintf("%s: %s %s: %v", ErrBatchFailed, result.Operation.Kind, result.Operation.Key, result.Err)
		}
	}
	return ErrBatchFailed.Error()
}

func (e BatchError) Is(target error) bool {
	return target == ErrBatchFailed
}

func (e BatchError) Unwrap() error {
	for _, result := range e.Results {
		if result.Err != nil {
			return result.Err
		}
	}
	return nil
}

func NewEntry(key string, value []byte, ttl time.Duration, version uint64) Entry {
	entry := Entry{Key: key, Value: append([]byte(nil), value...), Version: version}
	if ttl > 0 {
		entry.ExpiresAt = time.Now().Add(ttl).UnixNano()
	}
	return entry
}

func NewGet(key string) Operation {
	return Operation{Kind: OperationGet, Key: key}
}

func NewPut(key string, value []byte, options ...WriteOption) Operation {
	return Operation{Kind: OperationPut, Key: key, Value: append([]byte(nil), value...), Options: NewWriteOptions(options...)}
}

func NewDelete(key string, options ...WriteOption) Operation {
	return Operation{Kind: OperationDelete, Key: key, Options: NewWriteOptions(options...)}
}

func (e Entry) Expired(now time.Time) bool {
	return e.ExpiresAt > 0 && now.UnixNano() >= e.ExpiresAt
}

func (e Entry) Clone() Entry {
	e.Value = append([]byte(nil), e.Value...)
	return e
}

func (op Operation) Clone() Operation {
	op.Value = append([]byte(nil), op.Value...)
	return op
}

func (r Result) Clone() Result {
	r.Operation = r.Operation.Clone()
	r.Entry = r.Entry.Clone()
	return r
}
