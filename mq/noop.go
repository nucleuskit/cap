package mq

import (
	"context"
	"errors"
)

var (
	ErrNotConfigured = errors.New("mq not configured")
	ErrNoMessage     = errors.New("mq message not found")
)

type noopMQ struct{}

func NewNoop() *noopMQ {
	return &noopMQ{}
}

func (noopMQ) Publish(context.Context, Message) error {
	return ErrNotConfigured
}

func (noopMQ) PublishBatch(context.Context, ...Message) ([]PublishResult, error) {
	return nil, ErrNotConfigured
}

func (noopMQ) Consume(context.Context) (<-chan Delivery, error) {
	return nil, ErrNotConfigured
}

func (noopMQ) Subscribe(context.Context, Subscription) (<-chan Delivery, error) {
	return nil, ErrNotConfigured
}
