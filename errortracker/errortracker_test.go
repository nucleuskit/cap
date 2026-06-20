package errortracker

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNoopErrorTrackerImplementsProtocol(t *testing.T) {
	tracker := NewNoop()
	var reporter Reporter = tracker
	var flusher Flusher = tracker

	err := reporter.Capture(context.Background(), Event{
		Error:      errors.New("boom"),
		Operation:  "GET /users",
		Severity:   SeverityError,
		Tags:       map[string]string{"service": "demo"},
		Extra:      map[string]any{"attempt": 1},
		OccurredAt: time.Unix(1, 0),
	})
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
	if err := flusher.Flush(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Flush, got %v", err)
	}
}

func TestEventCloneDoesNotAliasMaps(t *testing.T) {
	event := Event{Tags: map[string]string{"a": "b"}, Extra: map[string]any{"n": 1}}
	clone := event.Clone()
	clone.Tags["a"] = "changed"
	clone.Extra["n"] = 2

	if event.Tags["a"] != "b" || event.Extra["n"] != 1 {
		t.Fatalf("expected clone to avoid aliasing, got %#v", event)
	}
}
