package auth

import (
	"context"
	"errors"
)

var ErrNotConfigured = errors.New("auth not configured")

type noopAuth struct{}

func NewNoop() *noopAuth {
	return &noopAuth{}
}

func (noopAuth) Authenticate(context.Context, Credentials) (Principal, error) {
	return Principal{}, ErrNotConfigured
}

func (noopAuth) Authorize(context.Context, Principal, Permission) error {
	return ErrNotConfigured
}
