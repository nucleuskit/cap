package httpclient

import (
	"context"
	"time"

	captransport "github.com/nucleuskit/cap/transport"
)

type TransportTargetPolicy func(context.Context, captransport.Target) (captransport.Target, error)

type Options struct {
	BaseURL               string
	Timeout               time.Duration
	UserAgent             string
	Retry                 RetryPolicy
	Headers               map[string]string
	TraceNames            []string
	TransportDialer       captransport.Dialer
	TransportTargetPolicy TransportTargetPolicy
	Hooks                 []Hook
}

type Option func(*Options)

func WithTimeout(timeout time.Duration) Option {
	return func(options *Options) {
		options.Timeout = timeout
	}
}

func WithUserAgent(userAgent string) Option {
	return func(options *Options) {
		options.UserAgent = userAgent
	}
}

func WithBaseURL(baseURL string) Option {
	return func(options *Options) {
		options.BaseURL = baseURL
	}
}

func WithRetry(policy RetryPolicy) Option {
	return func(options *Options) {
		options.Retry = policy
	}
}

func WithHeader(key, value string) Option {
	return func(options *Options) {
		if options.Headers == nil {
			options.Headers = map[string]string{}
		}
		options.Headers[key] = value
	}
}

func WithTraceHeader(name string) Option {
	return func(options *Options) {
		options.TraceNames = append(options.TraceNames, name)
	}
}

func WithTransportDialer(dialer captransport.Dialer) Option {
	return func(options *Options) {
		options.TransportDialer = dialer
	}
}

func WithTransportTargetPolicy(policy TransportTargetPolicy) Option {
	return func(options *Options) {
		options.TransportTargetPolicy = policy
	}
}

func WithHooks(hooks ...Hook) Option {
	return func(options *Options) {
		options.Hooks = append(options.Hooks, hooks...)
	}
}

func NewOptions(options ...Option) Options {
	values := Options{Timeout: 5 * time.Second}
	for _, option := range options {
		option(&values)
	}
	if values.Headers != nil {
		headers := make(map[string]string, len(values.Headers))
		for key, value := range values.Headers {
			headers[key] = value
		}
		values.Headers = headers
	}
	values.TraceNames = append([]string(nil), values.TraceNames...)
	values.Hooks = append([]Hook(nil), values.Hooks...)
	return values
}
