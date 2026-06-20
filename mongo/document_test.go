package mongo

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDocumentCloneIsIndependent(t *testing.T) {
	doc := NewDocument("42", map[string]any{
		"name":  "ada",
		"bytes": []byte("hello"),
		"nested": map[string]any{
			"role": "admin",
		},
	}, time.Minute)
	doc.Metadata = map[string]string{"source": "test"}

	clone := doc.Clone()
	clone.Fields["name"] = "grace"
	clone.Fields["nested"].(map[string]any)["role"] = "user"
	clone.Fields["bytes"].([]byte)[0] = 'H'
	clone.Metadata["source"] = "clone"

	if doc.Fields["name"] != "ada" ||
		doc.Fields["nested"].(map[string]any)["role"] != "admin" ||
		string(doc.Fields["bytes"].([]byte)) != "hello" ||
		doc.Metadata["source"] != "test" {
		t.Fatalf("clone mutated original: %#v", doc)
	}
	if !doc.Expired(time.Unix(0, doc.ExpiresAt+1)) {
		t.Fatal("expected document to expire after ExpiresAt")
	}
}

func TestMatchUsesIDVersionAndFields(t *testing.T) {
	doc := Document{ID: "u1", Version: 3, Fields: map[string]any{"name": "ada"}}
	if !Match(doc, Filter{"_id": "u1", "_version": int64(3), "name": "ada"}) {
		t.Fatal("expected filter to match document")
	}
	if Match(doc, Filter{"name": "grace"}) {
		t.Fatal("expected mismatched field to fail")
	}
}

func TestWriteOptions(t *testing.T) {
	options := NewWriteOptions(WithExpectedVersion(7), WithUpsert(true), WithTTL(time.Second))
	if options.ExpectedVersion != 7 || !options.Upsert || options.TTL != time.Second {
		t.Fatalf("unexpected write options: %#v", options)
	}
}

func TestNoopStoreReturnsNotConfigured(t *testing.T) {
	store := NewNoop()
	if _, err := store.Get(context.Background(), "users", "u1"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected not configured, got %v", err)
	}
}
