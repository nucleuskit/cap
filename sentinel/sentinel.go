package sentinel

import "context"

type Attribute struct {
	Key   string
	Value any
}

type Resource struct {
	Name       string
	Priority   int
	Attributes []Attribute
}

type Policy struct {
	Resource    string
	MaxInFlight int
	FailClosed  bool
}

type Guard interface {
	Done(error)
}

type Permit interface {
	Release()
}

type Breaker interface {
	Allow(ctx context.Context, resource Resource) (Guard, error)
}

type Limiter interface {
	Acquire(ctx context.Context, resource Resource) (Permit, error)
}

func String(key string, value string) Attribute {
	return Attribute{Key: key, Value: value}
}

func Any(key string, value any) Attribute {
	return Attribute{Key: key, Value: value}
}
