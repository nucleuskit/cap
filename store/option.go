package store

import "time"

type Options struct {
	Namespace    string
	Capacity     int
	DefaultTTL   time.Duration
	CacheAside   bool
	Singleflight bool
}

type Option func(*Options)

func WithNamespace(namespace string) Option {
	return func(options *Options) {
		options.Namespace = namespace
	}
}

func WithCapacity(capacity int) Option {
	return func(options *Options) {
		options.Capacity = capacity
	}
}

func WithDefaultTTL(ttl time.Duration) Option {
	return func(options *Options) {
		options.DefaultTTL = ttl
	}
}

func WithCacheAside(enabled bool) Option {
	return func(options *Options) {
		options.CacheAside = enabled
	}
}

func WithSingleflight(enabled bool) Option {
	return func(options *Options) {
		options.Singleflight = enabled
	}
}

func NewOptions(options ...Option) Options {
	values := Options{}
	for _, option := range options {
		option(&values)
	}
	return values
}
