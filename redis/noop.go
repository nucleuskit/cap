package redis

import (
	"context"
	"errors"
	"time"
)

var ErrNotConfigured = errors.New("redis not configured")

type noopClient struct{}

func NewNoop() *noopClient {
	return &noopClient{}
}

func (noopClient) Get(context.Context, string) ([]byte, error) {
	return nil, ErrNotConfigured
}

func (noopClient) MGet(context.Context, ...string) (map[string][]byte, error) {
	return nil, ErrNotConfigured
}

func (noopClient) Set(context.Context, string, []byte, time.Duration) error {
	return ErrNotConfigured
}

func (noopClient) MSet(context.Context, map[string][]byte, time.Duration) error {
	return ErrNotConfigured
}

func (noopClient) Delete(context.Context, string) error {
	return ErrNotConfigured
}

func (noopClient) Pipeline(context.Context, ...Command) ([]Result, error) {
	return nil, ErrNotConfigured
}
