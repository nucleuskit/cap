package lock

import (
	"context"
	"errors"
	"time"
)

var ErrLockNotHeld = errors.New("lock not held")

type Lock interface {
	Key() string
	Token() string
	Extend(ctx context.Context, ttl time.Duration) error
	Release(ctx context.Context) error
}

type Locker interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)
}
