package sentinel

type Options struct {
	Namespace  string
	FailClosed bool
	Policies   []Policy
}

type Option func(*Options)

func WithNamespace(namespace string) Option {
	return func(options *Options) {
		options.Namespace = namespace
	}
}

func WithFailClosed(failClosed bool) Option {
	return func(options *Options) {
		options.FailClosed = failClosed
	}
}

func WithPolicy(policy Policy) Option {
	return func(options *Options) {
		options.Policies = append(options.Policies, policy)
	}
}

func NewOptions(options ...Option) Options {
	values := Options{}
	for _, option := range options {
		option(&values)
	}
	return values
}
