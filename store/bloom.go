package store

import (
	"context"
	"errors"
	"fmt"
)

const DefaultBloomFalsePositiveRate = 0.01

var ErrInvalidBloomOptions = errors.New("invalid bloom options")

type BloomFilter interface {
	Add(ctx context.Context, key string) error
	Contains(ctx context.Context, key string) (bool, error)
	TestAndAdd(ctx context.Context, key string) (bool, error)
}

type ResettableBloomFilter interface {
	Reset(ctx context.Context) error
}

type BloomStatsProvider interface {
	Stats(ctx context.Context) (BloomStats, error)
}

type BloomOptions struct {
	Namespace         string
	Capacity          uint64
	FalsePositiveRate float64
	Bits              uint64
	Hashes            uint64
	Seed              uint64
}

type BloomStats struct {
	Namespace         string
	Capacity          uint64
	FalsePositiveRate float64
	Bits              uint64
	Hashes            uint64
	Added             uint64
}

type BloomOption func(*BloomOptions)

func WithBloomNamespace(namespace string) BloomOption {
	return func(options *BloomOptions) {
		options.Namespace = namespace
	}
}

func WithBloomCapacity(capacity uint64) BloomOption {
	return func(options *BloomOptions) {
		options.Capacity = capacity
	}
}

func WithBloomFalsePositiveRate(rate float64) BloomOption {
	return func(options *BloomOptions) {
		options.FalsePositiveRate = rate
	}
}

func WithBloomBits(bits uint64) BloomOption {
	return func(options *BloomOptions) {
		options.Bits = bits
	}
}

func WithBloomHashes(hashes uint64) BloomOption {
	return func(options *BloomOptions) {
		options.Hashes = hashes
	}
}

func WithBloomSeed(seed uint64) BloomOption {
	return func(options *BloomOptions) {
		options.Seed = seed
	}
}

func NewBloomOptions(options ...BloomOption) BloomOptions {
	values := BloomOptions{}
	for _, option := range options {
		option(&values)
	}
	return values
}

func (o *BloomOptions) ApplyDefaults() {
	if o.FalsePositiveRate == 0 {
		o.FalsePositiveRate = DefaultBloomFalsePositiveRate
	}
}

func (o BloomOptions) Validate() error {
	if o.Capacity == 0 {
		return fmt.Errorf("%w: capacity must be greater than zero", ErrInvalidBloomOptions)
	}
	if o.FalsePositiveRate <= 0 || o.FalsePositiveRate >= 1 {
		return fmt.Errorf("%w: false positive rate must be between zero and one", ErrInvalidBloomOptions)
	}
	if (o.Bits == 0) != (o.Hashes == 0) {
		return fmt.Errorf("%w: bits and hashes must be configured together", ErrInvalidBloomOptions)
	}
	return nil
}
