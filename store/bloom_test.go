package store

import (
	"context"
	"errors"
	"testing"
)

type bloomFilterStub struct {
	values map[string]struct{}
}

func (b *bloomFilterStub) Add(ctx context.Context, key string) error {
	if key == "" {
		return ErrInvalidBloomOptions
	}
	b.values[key] = struct{}{}
	return nil
}

func (b *bloomFilterStub) Contains(ctx context.Context, key string) (bool, error) {
	_, ok := b.values[key]
	return ok, nil
}

func (b *bloomFilterStub) TestAndAdd(ctx context.Context, key string) (bool, error) {
	existed, err := b.Contains(ctx, key)
	if err != nil {
		return false, err
	}
	if err := b.Add(ctx, key); err != nil {
		return false, err
	}
	return existed, nil
}

func TestBloomFilterProtocol(t *testing.T) {
	filter := &bloomFilterStub{values: map[string]struct{}{}}
	var bloom BloomFilter = filter

	existed, err := bloom.TestAndAdd(context.Background(), "user:1")
	if err != nil {
		t.Fatal(err)
	}
	if existed {
		t.Fatal("expected first TestAndAdd to report missing key")
	}

	existed, err = bloom.TestAndAdd(context.Background(), "user:1")
	if err != nil {
		t.Fatal(err)
	}
	if !existed {
		t.Fatal("expected second TestAndAdd to report existing key")
	}

	ok, err := bloom.Contains(context.Background(), "user:1")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected key to be present")
	}
}

func TestBloomOptionsValidateAndDefaults(t *testing.T) {
	options := NewBloomOptions(WithBloomNamespace("cache"), WithBloomCapacity(1000), WithBloomFalsePositiveRate(0.01), WithBloomSeed(42))
	if err := options.Validate(); err != nil {
		t.Fatalf("expected valid bloom options: %v", err)
	}
	if options.Namespace != "cache" {
		t.Fatalf("expected namespace cache, got %q", options.Namespace)
	}
	if options.Capacity != 1000 {
		t.Fatalf("expected capacity 1000, got %d", options.Capacity)
	}
	if options.FalsePositiveRate != 0.01 {
		t.Fatalf("expected false positive rate 0.01, got %f", options.FalsePositiveRate)
	}
	if options.Seed != 42 {
		t.Fatalf("expected seed 42, got %d", options.Seed)
	}

	defaulted := NewBloomOptions(WithBloomCapacity(64))
	defaulted.ApplyDefaults()
	if defaulted.FalsePositiveRate <= 0 || defaulted.FalsePositiveRate >= 1 {
		t.Fatalf("expected default false positive rate in (0, 1), got %f", defaulted.FalsePositiveRate)
	}
}

func TestBloomOptionsRejectInvalidParameters(t *testing.T) {
	tests := []struct {
		name    string
		options BloomOptions
	}{
		{name: "empty capacity", options: BloomOptions{FalsePositiveRate: 0.01}},
		{name: "zero false positive rate", options: BloomOptions{Capacity: 100, FalsePositiveRate: 0}},
		{name: "negative false positive rate", options: BloomOptions{Capacity: 100, FalsePositiveRate: -0.1}},
		{name: "one false positive rate", options: BloomOptions{Capacity: 100, FalsePositiveRate: 1}},
		{name: "bits without hashes", options: BloomOptions{Capacity: 100, FalsePositiveRate: 0.01, Bits: 1024}},
		{name: "hashes without bits", options: BloomOptions{Capacity: 100, FalsePositiveRate: 0.01, Hashes: 4}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.options.Validate(); !errors.Is(err, ErrInvalidBloomOptions) {
				t.Fatalf("expected ErrInvalidBloomOptions, got %v", err)
			}
		})
	}
}
