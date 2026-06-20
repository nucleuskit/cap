package transport

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewConfigClonesTransportPolicy(t *testing.T) {
	headers := map[string]string{"Proxy-Authorization": "token"}
	metadata := map[string]string{"service": "orders"}
	hook := DialHookFuncs{}
	cfg := NewConfig(
		WithNetwork("tcp4"),
		WithAddress("127.0.0.1:443"),
		WithServerName("orders.local"),
		WithTimeouts(TimeoutConfig{Dial: time.Second, KeepAlive: time.Minute, TLSHandshake: 2 * time.Second}),
		WithTLS(TLSConfig{Enabled: true, NextProtos: []string{"h2"}}),
		WithProxy(ProxyConfig{URL: "http://proxy.local:8080", Headers: headers}),
		WithMetadataMap(metadata),
		WithDialHooks(hook),
	)

	headers["Proxy-Authorization"] = "mutated"
	metadata["service"] = "mutated"

	if cfg.Network != "tcp4" || cfg.Address != "127.0.0.1:443" || cfg.ServerName != "orders.local" {
		t.Fatalf("unexpected endpoint config: %#v", cfg)
	}
	if cfg.Timeout.Dial != time.Second || cfg.Timeout.KeepAlive != time.Minute || cfg.Timeout.TLSHandshake != 2*time.Second {
		t.Fatalf("unexpected timeout config: %#v", cfg.Timeout)
	}
	if !cfg.TLS.Enabled || len(cfg.TLS.NextProtos) != 1 || cfg.TLS.NextProtos[0] != "h2" {
		t.Fatalf("unexpected tls config: %#v", cfg.TLS)
	}
	if cfg.Proxy.Headers["Proxy-Authorization"] != "token" {
		t.Fatalf("proxy headers were not cloned: %#v", cfg.Proxy.Headers)
	}
	if cfg.Metadata["service"] != "orders" {
		t.Fatalf("metadata was not cloned: %#v", cfg.Metadata)
	}
	if len(cfg.Hooks) != 1 {
		t.Fatalf("expected one hook, got %d", len(cfg.Hooks))
	}
}

func TestTargetCloneClonesTLSAndMetadata(t *testing.T) {
	target := Target{
		Address:  "example.com:443",
		Metadata: Metadata{"route": "search"},
		TLS:      &TLSConfig{Enabled: true, NextProtos: []string{"h2"}},
	}
	clone := target.Clone()
	target.Metadata["route"] = "mutated"
	target.TLS.NextProtos[0] = "http/1.1"

	if clone.Metadata["route"] != "search" {
		t.Fatalf("metadata was not cloned: %#v", clone.Metadata)
	}
	if clone.TLS.NextProtos[0] != "h2" {
		t.Fatalf("tls config was not cloned: %#v", clone.TLS)
	}
}

func TestDialHookFuncsCloneEvents(t *testing.T) {
	event := DialEvent{Metadata: Metadata{"key": "value"}}
	hook := DialHookFuncs{
		Before: func(ctx context.Context, event DialEvent) context.Context {
			event.Metadata["key"] = "before"
			return ctx
		},
		After: func(ctx context.Context, event DialEvent) {
			event.Metadata["key"] = "after"
		},
	}

	ctx := hook.BeforeDial(context.Background(), event)
	hook.AfterDial(ctx, event)

	if event.Metadata["key"] != "value" {
		t.Fatalf("event metadata was mutated by hook: %#v", event.Metadata)
	}
}

func TestNoopDialerReturnsNotConfigured(t *testing.T) {
	dialer := NewNoop()
	if _, err := dialer.DialContext(context.Background(), Target{Address: "127.0.0.1:1"}); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
	if stats := dialer.Stats(); stats != (Stats{}) {
		t.Fatalf("expected zero stats, got %#v", stats)
	}
}

func TestBackoffPolicyDurationCapsAndMultiplies(t *testing.T) {
	policy := BackoffPolicy{
		Initial:    100 * time.Millisecond,
		Max:        250 * time.Millisecond,
		Multiplier: 2,
	}
	if got := policy.Duration(1); got != 100*time.Millisecond {
		t.Fatalf("attempt 1 = %s", got)
	}
	if got := policy.Duration(2); got != 200*time.Millisecond {
		t.Fatalf("attempt 2 = %s", got)
	}
	if got := policy.Duration(3); got != 250*time.Millisecond {
		t.Fatalf("attempt 3 should cap at 250ms, got %s", got)
	}
}

func TestSleepBackoffHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := SleepBackoff(ctx, BackoffPolicy{Initial: time.Second}, 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}
