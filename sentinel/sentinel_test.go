package sentinel

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNoopSentinelImplementsBreakerAndLimiter(t *testing.T) {
	noop := NewNoop()
	var breaker Breaker = noop
	var limiter Limiter = noop

	_, err := breaker.Allow(context.Background(), Resource{Name: "downstream"})
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Breaker.Allow, got %v", err)
	}

	_, err = limiter.Acquire(context.Background(), Resource{Name: "orders"})
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Limiter.Acquire, got %v", err)
	}
}

func TestSentinelOptions(t *testing.T) {
	options := NewOptions(WithNamespace("api"), WithFailClosed(true), WithPolicy(Policy{Resource: "orders", MaxInFlight: 2}))
	if options.Namespace != "api" {
		t.Fatalf("expected namespace api, got %q", options.Namespace)
	}
	if !options.FailClosed {
		t.Fatal("expected fail closed")
	}
	if len(options.Policies) != 1 || options.Policies[0].Resource != "orders" {
		t.Fatalf("expected orders policy, got %#v", options.Policies)
	}
}

func TestRateLimiterRejectsOverWindowLimit(t *testing.T) {
	limiter := NewRateLimiter(RatePolicy{Resource: "orders", Limit: 1, Window: time.Minute})
	permit, err := limiter.Acquire(context.Background(), Resource{Name: "orders"})
	if err != nil {
		t.Fatal(err)
	}
	permit.Release()
	if _, err := limiter.Acquire(context.Background(), Resource{Name: "orders"}); !errors.Is(err, ErrRejected) {
		t.Fatalf("expected ErrRejected, got %v", err)
	}
}

func TestCircuitBreakerOpensAfterFailures(t *testing.T) {
	breaker := NewCircuitBreaker(BreakerPolicy{Resource: "orders", FailureThreshold: 1, OpenTimeout: time.Minute})
	guard, err := breaker.Allow(context.Background(), Resource{Name: "orders"})
	if err != nil {
		t.Fatal(err)
	}
	guard.Done(errors.New("downstream failed"))
	if _, err := breaker.Allow(context.Background(), Resource{Name: "orders"}); !errors.Is(err, ErrRejected) {
		t.Fatalf("expected ErrRejected, got %v", err)
	}
}

func TestAdaptiveShedderRejectsWhenCPUAndInflightAreHigh(t *testing.T) {
	shedder := NewAdaptiveShedder(ShedderPolicy{
		Resource:     "orders",
		MaxInFlight:  1,
		CPUThreshold: 0.5,
		CPU:          func() float64 { return 0.9 },
	})
	permit, err := shedder.Acquire(context.Background(), Resource{Name: "orders"})
	if err != nil {
		t.Fatal(err)
	}
	defer permit.Release()

	if _, err := shedder.Acquire(context.Background(), Resource{Name: "orders"}); !errors.Is(err, ErrRejected) {
		t.Fatalf("expected ErrRejected, got %v", err)
	}
}

func TestPriorityShedderRejectsLowPriorityWhenSaturated(t *testing.T) {
	shedder := NewPriorityShedder(PriorityPolicy{Resource: "orders", MaxInFlight: 1, MinPriority: 5})
	permit, err := shedder.Acquire(context.Background(), Resource{Name: "orders", Priority: 9})
	if err != nil {
		t.Fatal(err)
	}
	defer permit.Release()

	if _, err := shedder.Acquire(context.Background(), Resource{Name: "orders", Priority: 1}); !errors.Is(err, ErrRejected) {
		t.Fatalf("expected low priority request to be rejected, got %v", err)
	}
	if permit, err := shedder.Acquire(context.Background(), Resource{Name: "orders", Priority: 9}); err != nil {
		t.Fatalf("expected high priority request to pass under pressure, got %v", err)
	} else {
		permit.Release()
	}
}

func TestRollingWindowAggregatesRecentSamples(t *testing.T) {
	window := NewRollingWindow(time.Minute, 4)
	window.Add(10*time.Millisecond, nil)
	window.Add(20*time.Millisecond, errors.New("failed"))

	sample := window.Snapshot()
	if sample.Count != 2 || sample.Failures != 1 || sample.MinDuration != 10*time.Millisecond {
		t.Fatalf("unexpected sample: %#v", sample)
	}
}
