package kv

import "time"

type WriteOptions struct {
	TTL             time.Duration
	ExpectedVersion uint64
	MatchVersion    bool
}

type WriteOption func(*WriteOptions)

func NewWriteOptions(options ...WriteOption) WriteOptions {
	var values WriteOptions
	for _, option := range options {
		if option != nil {
			option(&values)
		}
	}
	return values
}

func WithTTL(ttl time.Duration) WriteOption {
	return func(options *WriteOptions) {
		options.TTL = ttl
	}
}

func WithExpectedVersion(version uint64) WriteOption {
	return func(options *WriteOptions) {
		options.ExpectedVersion = version
		options.MatchVersion = true
	}
}
