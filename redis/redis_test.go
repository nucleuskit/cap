package redis

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNoopClientImplementsRedisProtocol(t *testing.T) {
	client := NewNoop()
	var _ Client = client

	if _, err := client.Get(context.Background(), "key"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Get, got %v", err)
	}
	if err := client.Set(context.Background(), "key", []byte("value"), time.Minute); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Set, got %v", err)
	}
	if err := client.Delete(context.Background(), "key"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Delete, got %v", err)
	}
	if _, err := client.Pipeline(context.Background(), Command{Name: "get", Key: "key"}); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Pipeline, got %v", err)
	}
}

func TestRedisOptions(t *testing.T) {
	hook := OperationHookFuncs{}
	options := NewOptions(
		WithAddress("127.0.0.1:6379"),
		WithNamespace("demo"),
		WithDatabase(2),
		WithCluster("127.0.0.1:7001", "127.0.0.1:7002"),
		WithPool(PoolConfig{Size: 32}),
		WithRetry(RetryConfig{MaxAttempts: 3}),
		WithTimeouts(TimeoutConfig{Read: time.Second}),
		WithTLS(TLSConfig{Enabled: true, ServerName: "redis.local"}),
		WithOperationHooks(hook),
	)
	if options.Address != "127.0.0.1:6379" {
		t.Fatalf("expected redis address, got %q", options.Address)
	}
	if options.Namespace != "demo" {
		t.Fatalf("expected namespace demo, got %q", options.Namespace)
	}
	if options.Database != 2 || len(options.Addrs) != 2 || !options.Cluster.Enabled {
		t.Fatalf("expected redis topology options, got %#v", options)
	}
	if options.Pool.Size != 32 || options.Retry.MaxAttempts != 3 || options.Timeout.Read != time.Second || !options.TLS.Enabled {
		t.Fatalf("expected redis policy options, got %#v", options)
	}
	if len(options.Hooks) != 1 {
		t.Fatalf("expected hook option")
	}
}

func TestRedisConfigAndPipelineError(t *testing.T) {
	cfg := NewConfig(
		WithAddress("127.0.0.1:6379"),
		WithCluster("127.0.0.1:7001"),
		WithNamespace("demo"),
	)
	if cfg.Mode != ModeCluster || len(cfg.Cluster.Addrs) != 1 || cfg.Namespace != "demo" {
		t.Fatalf("unexpected config: %#v", cfg)
	}

	results := []Result{{Command: Command{Name: "GET", Key: "missing"}, Err: ErrNotConfigured}}
	err := PipelineError{Results: results}
	if !errors.Is(err, ErrPipelineFailed) {
		t.Fatalf("expected pipeline error to match ErrPipelineFailed")
	}
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected pipeline error to unwrap command error")
	}
}
