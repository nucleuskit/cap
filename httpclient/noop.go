package httpclient

import (
	"context"
	"errors"
)

var ErrNotConfigured = errors.New("http client not configured")

type noopClient struct{}

func NewNoop() Client {
	return noopClient{}
}

func (noopClient) Do(context.Context, Request) (Response, error) {
	return Response{}, ErrNotConfigured
}
