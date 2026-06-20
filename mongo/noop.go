package mongo

import (
	"context"
	"errors"
)

var ErrNotConfigured = errors.New("document store not configured")

type noopStore struct{}

func NewNoop() *noopStore {
	return &noopStore{}
}

func (noopStore) Insert(context.Context, string, Document, ...WriteOption) (Document, error) {
	return Document{}, ErrNotConfigured
}

func (noopStore) Get(context.Context, string, string) (Document, error) {
	return Document{}, ErrNotConfigured
}

func (noopStore) Find(context.Context, Query) ([]Document, error) {
	return nil, ErrNotConfigured
}

func (noopStore) Replace(context.Context, string, Document, ...WriteOption) (Document, error) {
	return Document{}, ErrNotConfigured
}

func (noopStore) Update(context.Context, string, string, Patch, ...WriteOption) (Document, error) {
	return Document{}, ErrNotConfigured
}

func (noopStore) Delete(context.Context, string, string) error {
	return ErrNotConfigured
}
