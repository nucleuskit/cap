package lock

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNoopLockerImplementsLockProtocol(t *testing.T) {
	locker := NewNoop()
	var _ Locker = locker

	_, err := locker.Acquire(context.Background(), "jobs/rebuild", time.Second)
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Acquire, got %v", err)
	}
}

func TestLockOptions(t *testing.T) {
	options := NewOptions(WithNamespace("jobs"), WithTTL(3*time.Second))
	if options.Namespace != "jobs" {
		t.Fatalf("expected namespace jobs, got %q", options.Namespace)
	}
	if options.TTL != 3*time.Second {
		t.Fatalf("expected ttl 3s, got %s", options.TTL)
	}
}
