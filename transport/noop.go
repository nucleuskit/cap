package transport

import (
	"context"
	"net"
)

type noopDialer struct{}

func NewNoop() *noopDialer {
	return &noopDialer{}
}

func (noopDialer) DialContext(context.Context, Target) (net.Conn, error) {
	return nil, ErrNotConfigured
}

func (noopDialer) Stats() Stats {
	return Stats{}
}

func (noopDialer) Close() error {
	return nil
}
