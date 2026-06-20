package kv

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEntryCloneDoesNotAliasValue(t *testing.T) {
	entry := Entry{Key: "profile", Value: []byte("alice"), Version: 3, ExpiresAt: time.Now().Add(time.Minute).UnixNano()}
	clone := entry.Clone()
	clone.Value[0] = 'A'

	if string(entry.Value) != "alice" {
		t.Fatalf("entry value was mutated: %q", entry.Value)
	}
	if clone.Version != 3 {
		t.Fatalf("expected clone version 3, got %d", clone.Version)
	}
}

func TestWriteOptionsCaptureTTLAndCASVersion(t *testing.T) {
	options := NewWriteOptions(WithTTL(time.Minute), WithExpectedVersion(7))

	if options.TTL != time.Minute {
		t.Fatalf("expected ttl one minute, got %s", options.TTL)
	}
	if !options.MatchVersion {
		t.Fatal("expected version matching to be enabled")
	}
	if options.ExpectedVersion != 7 {
		t.Fatalf("expected version 7, got %d", options.ExpectedVersion)
	}
}

func TestNoopStoreReturnsNotConfigured(t *testing.T) {
	store := NewNoop()
	var _ Store = store

	if _, err := store.Get(context.Background(), "key"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Get, got %v", err)
	}
	if _, err := store.Put(context.Background(), "key", []byte("value")); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Put, got %v", err)
	}
	if err := store.Delete(context.Background(), "key"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Delete, got %v", err)
	}
	if _, err := store.Batch(context.Background(), NewGet("key")); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Batch, got %v", err)
	}
}
