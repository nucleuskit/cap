package lock

import (
	"context"
	"errors"
	"time"
)

var ErrNotConfigured = errors.New("lock not configured")

type noopLocker struct{}

func NewNoop() *noopLocker {
	return &noopLocker{}
}

func (noopLocker) Acquire(context.Context, string, time.Duration) (Lock, error) {
	return nil, ErrNotConfigured
}
