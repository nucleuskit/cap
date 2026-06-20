package mongo

import "time"

type WriteOptions struct {
	ExpectedVersion int64
	Upsert          bool
	TTL             time.Duration
}

type WriteOption func(*WriteOptions)

func WithExpectedVersion(version int64) WriteOption {
	return func(options *WriteOptions) {
		options.ExpectedVersion = version
	}
}

func WithUpsert(enabled bool) WriteOption {
	return func(options *WriteOptions) {
		options.Upsert = enabled
	}
}

func WithTTL(ttl time.Duration) WriteOption {
	return func(options *WriteOptions) {
		options.TTL = ttl
	}
}

func NewWriteOptions(options ...WriteOption) WriteOptions {
	var values WriteOptions
	for _, option := range options {
		if option != nil {
			option(&values)
		}
	}
	return values
}
