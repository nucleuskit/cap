package errortracker

import (
	"context"
	"errors"
	"time"
)

var ErrNotConfigured = errors.New("error tracker not configured")

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
	SeverityFatal   Severity = "fatal"
)

type Event struct {
	Error      error
	Message    string
	Operation  string
	Severity   Severity
	Tags       map[string]string
	Extra      map[string]any
	OccurredAt time.Time
}

type Reporter interface {
	Capture(context.Context, Event) error
}

type Flusher interface {
	Flush(context.Context) error
}

type ReporterFunc func(context.Context, Event) error

func (fn ReporterFunc) Capture(ctx context.Context, event Event) error {
	if fn == nil {
		return ErrNotConfigured
	}
	return fn(ctx, event.Clone())
}

type noopTracker struct{}

func NewNoop() *noopTracker {
	return &noopTracker{}
}

func (noopTracker) Capture(context.Context, Event) error {
	return ErrNotConfigured
}

func (noopTracker) Flush(context.Context) error {
	return ErrNotConfigured
}

func (e Event) Clone() Event {
	e.Tags = cloneStringMap(e.Tags)
	e.Extra = cloneAnyMap(e.Extra)
	return e
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneAnyMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
