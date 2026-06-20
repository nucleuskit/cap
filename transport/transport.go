package transport

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"net"
	"time"
)

var (
	ErrNotConfigured = errors.New("transport not configured")
	ErrClosed        = errors.New("transport dialer closed")
	ErrMissingTarget = errors.New("transport target address is empty")
)

type Network string

const (
	NetworkTCP  Network = "tcp"
	NetworkTCP4 Network = "tcp4"
	NetworkTCP6 Network = "tcp6"
	NetworkUDP  Network = "udp"
	NetworkUnix Network = "unix"
)

type Metadata map[string]string

type TimeoutConfig struct {
	Dial         time.Duration
	KeepAlive    time.Duration
	TLSHandshake time.Duration
}

type TLSConfig struct {
	Enabled            bool
	ServerName         string
	InsecureSkipVerify bool
	MinVersion         string
	MaxVersion         string
	NextProtos         []string
}

type ProxyConfig struct {
	URL             string
	FromEnvironment bool
	Headers         map[string]string
}

type BackoffPolicy struct {
	Initial    time.Duration
	Max        time.Duration
	Multiplier float64
	Jitter     float64
}

type Target struct {
	Network    string
	Address    string
	ServerName string
	TLS        *TLSConfig
	Metadata   Metadata
}

type Config struct {
	Network    string
	Address    string
	ServerName string
	Timeout    TimeoutConfig
	TLS        TLSConfig
	Proxy      ProxyConfig
	Metadata   Metadata
	Hooks      []DialHook
}

type Stats struct {
	Dials         int64
	Successes     int64
	Errors        int64
	TLSHandshakes int64
	ProxyDials    int64
	LastNetwork   string
	LastAddress   string
	LastProxy     string
	LastDuration  time.Duration
	LastError     string
}

type DialEvent struct {
	Network   string
	Address   string
	TLS       bool
	Proxy     string
	Metadata  Metadata
	StartedAt time.Time
	Duration  time.Duration
	Err       error
}

type DialHook interface {
	BeforeDial(context.Context, DialEvent) context.Context
	AfterDial(context.Context, DialEvent)
}

type DialHookFuncs struct {
	Before func(context.Context, DialEvent) context.Context
	After  func(context.Context, DialEvent)
}

func (h DialHookFuncs) BeforeDial(ctx context.Context, event DialEvent) context.Context {
	if h.Before == nil {
		return ctx
	}
	next := h.Before(ctx, event.Clone())
	if next == nil {
		return ctx
	}
	return next
}

func (h DialHookFuncs) AfterDial(ctx context.Context, event DialEvent) {
	if h.After != nil {
		h.After(ctx, event.Clone())
	}
}

type Dialer interface {
	DialContext(context.Context, Target) (net.Conn, error)
}

type Statser interface {
	Stats() Stats
}

type Closer interface {
	Close() error
}

func (t Target) Clone() Target {
	t.Metadata = CloneMetadata(t.Metadata)
	if t.TLS != nil {
		tls := t.TLS.Clone()
		t.TLS = &tls
	}
	return t
}

func (c Config) Clone() Config {
	c.Metadata = CloneMetadata(c.Metadata)
	c.Proxy = c.Proxy.Clone()
	c.TLS = c.TLS.Clone()
	c.Hooks = append([]DialHook(nil), c.Hooks...)
	return c
}

func (c TLSConfig) Clone() TLSConfig {
	c.NextProtos = append([]string(nil), c.NextProtos...)
	return c
}

func (c ProxyConfig) Clone() ProxyConfig {
	c.Headers = CloneMetadata(c.Headers)
	return c
}

func (s Stats) Clone() Stats {
	return s
}

func (e DialEvent) Clone() DialEvent {
	e.Metadata = CloneMetadata(e.Metadata)
	return e
}

func CloneMetadata(values map[string]string) Metadata {
	if len(values) == 0 {
		return nil
	}
	copied := make(Metadata, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

func MergeMetadata(values ...map[string]string) Metadata {
	var merged Metadata
	for _, value := range values {
		for key, item := range value {
			if merged == nil {
				merged = Metadata{}
			}
			merged[key] = item
		}
	}
	return merged
}

func DefaultNetwork(network string) string {
	if network == "" {
		return string(NetworkTCP)
	}
	return network
}

func (p BackoffPolicy) Duration(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	initial := p.Initial
	if initial <= 0 {
		initial = 100 * time.Millisecond
	}
	multiplier := p.Multiplier
	if multiplier <= 0 {
		multiplier = 2
	}
	value := float64(initial)
	if attempt > 1 {
		value = value * math.Pow(multiplier, float64(attempt-1))
	}
	duration := time.Duration(value)
	if p.Max > 0 && duration > p.Max {
		duration = p.Max
	}
	if p.Jitter > 0 {
		duration = withJitter(duration, p.Jitter)
	}
	return duration
}

func SleepBackoff(ctx context.Context, policy BackoffPolicy, attempt int) error {
	delay := policy.Duration(attempt)
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func withJitter(duration time.Duration, ratio float64) time.Duration {
	if ratio <= 0 || duration <= 0 {
		return duration
	}
	if ratio > 1 {
		ratio = 1
	}
	delta := int64(float64(duration) * ratio)
	if delta <= 0 {
		return duration
	}
	offset := rand.Int63n(delta*2+1) - delta
	return duration + time.Duration(offset)
}
