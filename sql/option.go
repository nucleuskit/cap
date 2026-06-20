package sql

type Options struct {
	Driver string
	Name   string
	Hooks  []QueryHook
}

type Option func(*Options)

func WithDriver(driver string) Option {
	return func(options *Options) {
		options.Driver = driver
	}
}

func WithName(name string) Option {
	return func(options *Options) {
		options.Name = name
	}
}

func WithQueryHooks(hooks ...QueryHook) Option {
	return func(options *Options) {
		options.Hooks = append(options.Hooks, hooks...)
	}
}

func NewOptions(options ...Option) Options {
	values := Options{}
	for _, option := range options {
		option(&values)
	}
	values.Hooks = append([]QueryHook(nil), values.Hooks...)
	return values
}
