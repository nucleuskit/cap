package transport

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestConnectionPolicyCloneClonesHooks(t *testing.T) {
	hook := ManagerHookFuncs{}
	policy := ConnectionPolicy{
		MaxAttempts: 3,
		Backoff:     BackoffPolicy{Initial: time.Millisecond, Max: time.Second, Multiplier: 2},
		Hooks:       []ManagerHook{hook},
	}

	clone := policy.Clone()
	policy.Hooks[0] = nil

	if clone.MaxAttempts != 3 {
		t.Fatalf("unexpected max attempts: %d", clone.MaxAttempts)
	}
	if clone.Backoff.Initial != time.Millisecond || clone.Backoff.Max != time.Second || clone.Backoff.Multiplier != 2 {
		t.Fatalf("unexpected backoff clone: %#v", clone.Backoff)
	}
	if len(clone.Hooks) != 1 || clone.Hooks[0] == nil {
		t.Fatalf("hooks were not cloned: %#v", clone.Hooks)
	}
}

func TestConnectionEventCloneClonesTarget(t *testing.T) {
	event := ConnectionEvent{
		State:   ConnectionStateRetrying,
		Attempt: 2,
		Target:  Target{Address: "127.0.0.1:80", Metadata: Metadata{"route": "search"}},
	}

	clone := event.Clone()
	event.Target.Metadata["route"] = "mutated"

	if clone.Target.Metadata["route"] != "search" {
		t.Fatalf("target metadata was not cloned: %#v", clone.Target.Metadata)
	}
	if clone.State != ConnectionStateRetrying || clone.Attempt != 2 {
		t.Fatalf("unexpected clone: %#v", clone)
	}
}

func TestManagerHookFuncsCloneEvents(t *testing.T) {
	event := ConnectionEvent{Target: Target{Metadata: Metadata{"key": "value"}}}
	hook := ManagerHookFuncs{
		OnEvent: func(event ConnectionEvent) {
			event.Target.Metadata["key"] = "mutated"
		},
	}

	hook.HandleConnectionEvent(event)

	if event.Target.Metadata["key"] != "value" {
		t.Fatalf("event target was mutated by hook: %#v", event.Target.Metadata)
	}
}

func TestConnectionManagerInterfaceShape(t *testing.T) {
	var _ ConnectionManager = managerShape{}
}

type managerShape struct{}

func (managerShape) Connect(context.Context, Target) (net.Conn, error) {
	return nil, nil
}

func (managerShape) State() ConnectionState {
	return ConnectionStateIdle
}

func (managerShape) Stats() ConnectionStats {
	return ConnectionStats{}
}

func (managerShape) Close() error {
	return nil
}

func (managerShape) Current() net.Conn {
	return nil
}
