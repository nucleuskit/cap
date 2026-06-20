package redis

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrPipelineFailed = errors.New("redis pipeline failed")

type Mode string

const (
	ModeStandalone Mode = "standalone"
	ModeCluster    Mode = "cluster"
)

type Command struct {
	Name  string
	Key   string
	Value []byte
	TTL   time.Duration
	Args  []any
}

type Result struct {
	Command Command
	Value   []byte
	Err     error
}

type PipelineError struct {
	Results []Result
}

func (e PipelineError) Error() string {
	if len(e.Results) == 0 {
		return ErrPipelineFailed.Error()
	}
	for _, result := range e.Results {
		if result.Err != nil {
			return fmt.Sprintf("%s: %s %s: %v", ErrPipelineFailed, result.Command.Name, result.Command.Key, result.Err)
		}
	}
	return ErrPipelineFailed.Error()
}

func (e PipelineError) Is(target error) bool {
	return target == ErrPipelineFailed
}

func (e PipelineError) Unwrap() error {
	for _, result := range e.Results {
		if result.Err != nil {
			return result.Err
		}
	}
	return nil
}

type Endpoint struct {
	Address  string
	Username string
	Password string
	Database int
}

type ClusterConfig struct {
	Enabled         bool
	Addrs           []string
	RouteByLatency  bool
	RouteRandomly   bool
	ReadOnly        bool
	MaxRedirects    int
	Username        string
	Password        string
	UseReplicasOnly bool
}

type PoolConfig struct {
	Size            int
	MinIdle         int
	MaxIdle         int
	MaxActive       int
	MaxLifetime     time.Duration
	IdleTimeout     time.Duration
	WaitTimeout     time.Duration
	ConnMaxIdleTime time.Duration
}

type RetryConfig struct {
	MaxAttempts int
	BackoffMin  time.Duration
	BackoffMax  time.Duration
}

type TimeoutConfig struct {
	Dial  time.Duration
	Read  time.Duration
	Write time.Duration
	Pool  time.Duration
}

type TLSConfig struct {
	Enabled            bool
	ServerName         string
	InsecureSkipVerify bool
	MinVersion         string
}

type Config struct {
	Mode      Mode
	Endpoint  Endpoint
	Cluster   ClusterConfig
	Pool      PoolConfig
	Retry     RetryConfig
	Timeout   TimeoutConfig
	TLS       TLSConfig
	Namespace string
}

type Stats struct {
	Commands  int64
	Hits      int64
	Misses    int64
	Sets      int64
	Deletes   int64
	Pipelines int64
	Errors    int64
}

type OperationEvent struct {
	Name         string
	Key          string
	Keys         []string
	CommandCount int
	StartedAt    time.Time
	Duration     time.Duration
	Err          error
}

type OperationHook interface {
	BeforeRedis(ctx context.Context, event OperationEvent) context.Context
	AfterRedis(ctx context.Context, event OperationEvent)
}

type OperationHookFuncs struct {
	Before func(ctx context.Context, event OperationEvent) context.Context
	After  func(ctx context.Context, event OperationEvent)
}

func (h OperationHookFuncs) BeforeRedis(ctx context.Context, event OperationEvent) context.Context {
	if h.Before == nil {
		return ctx
	}
	return h.Before(ctx, event.Clone())
}

func (h OperationHookFuncs) AfterRedis(ctx context.Context, event OperationEvent) {
	if h.After != nil {
		h.After(ctx, event.Clone())
	}
}

type Pipeline interface {
	Do(ctx context.Context, commands ...Command) ([]Result, error)
}

type Client interface {
	Get(ctx context.Context, key string) ([]byte, error)
	MGet(ctx context.Context, keys ...string) (map[string][]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	MSet(ctx context.Context, values map[string][]byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Pipeline(ctx context.Context, commands ...Command) ([]Result, error)
}

type Statser interface {
	Stats() Stats
}

func NewConfig(options ...Option) Config {
	values := NewOptions(options...)
	cfg := Config{
		Mode:      ModeStandalone,
		Namespace: values.Namespace,
		Endpoint: Endpoint{
			Address:  values.Address,
			Database: values.Database,
		},
		Cluster: values.Cluster,
		Pool:    values.Pool,
		Retry:   values.Retry,
		Timeout: values.Timeout,
		TLS:     values.TLS,
	}
	if len(values.Addrs) > 0 {
		cfg.Cluster.Addrs = append([]string(nil), values.Addrs...)
		cfg.Cluster.Enabled = true
		cfg.Mode = ModeCluster
	}
	return cfg
}

func (c Command) Clone() Command {
	c.Value = append([]byte(nil), c.Value...)
	c.Args = append([]any(nil), c.Args...)
	return c
}

func (r Result) Clone() Result {
	r.Command = r.Command.Clone()
	r.Value = append([]byte(nil), r.Value...)
	return r
}

func (s Stats) Clone() Stats {
	return s
}

func (e OperationEvent) Clone() OperationEvent {
	e.Keys = append([]string(nil), e.Keys...)
	return e
}
