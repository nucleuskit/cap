package sentinel

import (
	"context"
	"errors"
)

var ErrNotConfigured = errors.New("sentinel not configured")
var ErrRejected = errors.New("sentinel rejected")

type noopSentinel struct{}

func NewNoop() *noopSentinel {
	return &noopSentinel{}
}

func (noopSentinel) Allow(context.Context, Resource) (Guard, error) {
	return nil, ErrNotConfigured
}

func (noopSentinel) Acquire(context.Context, Resource) (Permit, error) {
	return nil, ErrNotConfigured
}
