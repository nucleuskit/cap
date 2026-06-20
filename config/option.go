package config

type Options struct {
	Source       string
	Path         string
	CachePath    string
	TemplatePath string
	Fallback     bool
	Priority     int
}

type Option func(*Options)

func WithSource(source string) Option {
	return func(options *Options) {
		options.Source = source
	}
}

func WithPath(path string) Option {
	return func(options *Options) {
		options.Path = path
	}
}

func WithCachePath(path string) Option {
	return func(options *Options) {
		options.CachePath = path
	}
}

func WithTemplatePath(path string) Option {
	return func(options *Options) {
		options.TemplatePath = path
	}
}

func WithFallback(enabled bool) Option {
	return func(options *Options) {
		options.Fallback = enabled
	}
}

func WithPriority(priority int) Option {
	return func(options *Options) {
		options.Priority = priority
	}
}

func NewOptions(options ...Option) Options {
	values := Options{Source: "file"}
	for _, option := range options {
		option(&values)
	}
	return values
}
