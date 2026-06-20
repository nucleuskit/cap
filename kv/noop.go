package kv

import (
	"context"
	"errors"
)

var ErrNotConfigured = errors.New("kv not configured")

type noopStore struct{}

func NewNoop() *noopStore {
	return &noopStore{}
}

func (noopStore) Get(context.Context, string) (Entry, error) {
	return Entry{}, ErrNotConfigured
}

func (noopStore) Put(context.Context, string, []byte, ...WriteOption) (Entry, error) {
	return Entry{}, ErrNotConfigured
}

func (noopStore) Delete(context.Context, string, ...WriteOption) error {
	return ErrNotConfigured
}

func (noopStore) Batch(context.Context, ...Operation) ([]Result, error) {
	return nil, ErrNotConfigured
}
