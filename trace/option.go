package trace

type Options struct {
	Service string
	Sampler string
}

type Option func(*Options)

func WithService(service string) Option {
	return func(options *Options) {
		options.Service = service
	}
}

func WithSampler(sampler string) Option {
	return func(options *Options) {
		options.Sampler = sampler
	}
}

func NewOptions(options ...Option) Options {
	values := Options{Sampler: "parentbased_traceidratio"}
	for _, option := range options {
		option(&values)
	}
	return values
}
