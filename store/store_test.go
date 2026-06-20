package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNoopStoreImplementsStoreProtocol(t *testing.T) {
	store := NewNoop()
	var _ Store = store

	if _, err := store.Get(context.Background(), "key"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Get, got %v", err)
	}
	if err := store.Set(context.Background(), Entry{Key: "key", Value: []byte("value")}); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Set, got %v", err)
	}
	if err := store.Delete(context.Background(), "key"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Delete, got %v", err)
	}
	if _, err := store.List(context.Background(), "prefix/"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from List, got %v", err)
	}
}

func TestStoreOptions(t *testing.T) {
	options := NewOptions(WithNamespace("cache"), WithCapacity(128), WithDefaultTTL(time.Minute), WithCacheAside(true), WithSingleflight(true))
	if options.Namespace != "cache" {
		t.Fatalf("expected namespace cache, got %q", options.Namespace)
	}
	if options.Capacity != 128 {
		t.Fatalf("expected capacity 128, got %d", options.Capacity)
	}
	if options.DefaultTTL != time.Minute {
		t.Fatalf("expected ttl one minute, got %s", options.DefaultTTL)
	}
	if !options.CacheAside || !options.Singleflight {
		t.Fatalf("expected cache-aside singleflight options")
	}
}

func TestEntryCloneAndExpiration(t *testing.T) {
	entry := NewEntry("key", []byte("value"), time.Nanosecond)
	entry.Metadata = map[string]string{"source": "test"}
	clone := entry.Clone()
	clone.Value[0] = 'V'
	clone.Metadata["source"] = "clone"
	if string(entry.Value) != "value" {
		t.Fatalf("entry value was mutated")
	}
	if entry.Metadata["source"] != "test" {
		t.Fatalf("entry metadata was mutated")
	}
	time.Sleep(time.Millisecond)
	if !entry.Expired(time.Now()) {
		t.Fatalf("expected entry to expire")
	}
}
