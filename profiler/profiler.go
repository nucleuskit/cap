package profiler

import (
	"context"
	"errors"
	"time"
)

var ErrNotConfigured = errors.New("profiler not configured")

type Type string

const (
	TypeCPU       Type = "cpu"
	TypeHeap      Type = "heap"
	TypeGoroutine Type = "goroutine"
	TypeMutex     Type = "mutex"
	TypeBlock     Type = "block"
)

type Session struct {
	ID        string
	Type      Type
	Duration  time.Duration
	Labels    map[string]string
	StartedAt time.Time
}

type Snapshot struct {
	Type       Type
	Provider   string
	CapturedAt time.Time
	Data       []byte
	Labels     map[string]string
}

type Controller interface {
	Start(context.Context, Session) error
	Stop(context.Context, string) error
}

type Snapshotter interface {
	Snapshot(context.Context, Type) (Snapshot, error)
}

type noopProfiler struct{}

func NewNoop() *noopProfiler {
	return &noopProfiler{}
}

func (noopProfiler) Start(context.Context, Session) error {
	return ErrNotConfigured
}

func (noopProfiler) Stop(context.Context, string) error {
	return ErrNotConfigured
}

func (noopProfiler) Snapshot(context.Context, Type) (Snapshot, error) {
	return Snapshot{}, ErrNotConfigured
}

func (s Session) Clone() Session {
	s.Labels = cloneLabels(s.Labels)
	return s
}

func (s Snapshot) Clone() Snapshot {
	s.Data = append([]byte(nil), s.Data...)
	s.Labels = cloneLabels(s.Labels)
	return s
}

func cloneLabels(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
