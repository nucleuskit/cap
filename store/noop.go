package store

import (
	"context"
	"errors"
)

var ErrNotConfigured = errors.New("store not configured")

type noopStore struct{}

func NewNoop() *noopStore {
	return &noopStore{}
}

func (noopStore) Get(context.Context, string) (Entry, error) {
	return Entry{}, ErrNotConfigured
}

func (noopStore) Set(context.Context, Entry) error {
	return ErrNotConfigured
}

func (noopStore) Add(context.Context, Entry) error {
	return ErrNotConfigured
}

func (noopStore) Delete(context.Context, string) error {
	return ErrNotConfigured
}

func (noopStore) List(context.Context, string) ([]Entry, error) {
	return nil, ErrNotConfigured
}
