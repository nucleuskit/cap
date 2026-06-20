package transport

import (
	"context"
	"net"
	"time"
)

type ConnectionState string

const (
	ConnectionStateIdle       ConnectionState = "idle"
	ConnectionStateConnecting ConnectionState = "connecting"
	ConnectionStateOpen       ConnectionState = "open"
	ConnectionStateRetrying   ConnectionState = "retrying"
	ConnectionStateClosed     ConnectionState = "closed"
	ConnectionStateFailed     ConnectionState = "failed"
)

type ConnectionPolicy struct {
	MaxAttempts int
	Backoff     BackoffPolicy
	Hooks       []ManagerHook
}

type ConnectionStats struct {
	Attempts    int64
	Successes   int64
	Failures    int64
	LastState   ConnectionState
	LastNetwork string
	LastAddress string
	LastAttempt int
	LastError   string
	UpdatedAt   time.Time
}

type ConnectionEvent struct {
	State   ConnectionState
	Attempt int
	Target  Target
	Err     error
	At      time.Time
}

type ManagerHook interface {
	HandleConnectionEvent(ConnectionEvent)
}

type ManagerHookFuncs struct {
	OnEvent func(ConnectionEvent)
}

func (h ManagerHookFuncs) HandleConnectionEvent(event ConnectionEvent) {
	if h.OnEvent != nil {
		h.OnEvent(event.Clone())
	}
}

type ConnectionManager interface {
	Connect(context.Context, Target) (net.Conn, error)
	State() ConnectionState
	Stats() ConnectionStats
	Current() net.Conn
	Close() error
}

func (p ConnectionPolicy) Clone() ConnectionPolicy {
	p.Hooks = append([]ManagerHook(nil), p.Hooks...)
	return p
}

func (s ConnectionStats) Clone() ConnectionStats {
	return s
}

func (e ConnectionEvent) Clone() ConnectionEvent {
	e.Target = e.Target.Clone()
	return e
}
