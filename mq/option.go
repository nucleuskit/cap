package mq

type Options struct {
	Broker       string
	Topic        string
	Group        string
	ClientID     string
	SessionID    string
	Callback     ProducerCallback
	Retry        RetryPolicy
	DeadLetter   DeadLetterPolicy
	ErrorHandler ErrorHandler
}

type Option func(*Options)

func WithBroker(broker string) Option {
	return func(options *Options) {
		options.Broker = broker
	}
}

func WithTopic(topic string) Option {
	return func(options *Options) {
		options.Topic = topic
	}
}

func WithGroup(group string) Option {
	return func(options *Options) {
		options.Group = group
	}
}

func WithClientID(clientID string) Option {
	return func(options *Options) {
		options.ClientID = clientID
	}
}

func WithSessionID(sessionID string) Option {
	return func(options *Options) {
		options.SessionID = sessionID
	}
}

func WithProducerCallback(callback ProducerCallback) Option {
	return func(options *Options) {
		options.Callback = callback
	}
}

func WithRetryPolicy(policy RetryPolicy) Option {
	return func(options *Options) {
		options.Retry = policy
	}
}

func WithDeadLetterPolicy(policy DeadLetterPolicy) Option {
	return func(options *Options) {
		options.DeadLetter = policy
	}
}

func WithErrorHandler(handler ErrorHandler) Option {
	return func(options *Options) {
		options.ErrorHandler = handler
	}
}

func NewOptions(options ...Option) Options {
	values := Options{}
	for _, option := range options {
		option(&values)
	}
	return values
}
