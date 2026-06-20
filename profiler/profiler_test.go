package profiler

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNoopProfilerImplementsProtocol(t *testing.T) {
	profiler := NewNoop()
	var controller Controller = profiler
	var snapshotter Snapshotter = profiler

	if err := controller.Start(context.Background(), Session{ID: "cpu", Type: TypeCPU, Duration: time.Second}); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Start, got %v", err)
	}
	if _, err := snapshotter.Snapshot(context.Background(), TypeHeap); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Snapshot, got %v", err)
	}
}

func TestSessionCloneDoesNotAliasLabels(t *testing.T) {
	session := Session{Labels: map[string]string{"service": "demo"}}
	clone := session.Clone()
	clone.Labels["service"] = "changed"
	if session.Labels["service"] != "demo" {
		t.Fatalf("expected labels to be cloned, got %#v", session.Labels)
	}
}
