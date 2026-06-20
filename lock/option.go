package lock

import "time"

type Options struct {
	Namespace string
	TTL       time.Duration
}

type Option func(*Options)

func WithNamespace(namespace string) Option {
	return func(options *Options) {
		options.Namespace = namespace
	}
}

func WithTTL(ttl time.Duration) Option {
	return func(options *Options) {
		options.TTL = ttl
	}
}

func NewOptions(options ...Option) Options {
	values := Options{}
	for _, option := range options {
		option(&values)
	}
	return values
}
