package auth

type Options struct {
	Issuer   string
	Audience string
}

type Option func(*Options)

func WithIssuer(issuer string) Option {
	return func(options *Options) {
		options.Issuer = issuer
	}
}

func WithAudience(audience string) Option {
	return func(options *Options) {
		options.Audience = audience
	}
}

func NewOptions(options ...Option) Options {
	values := Options{}
	for _, option := range options {
		option(&values)
	}
	return values
}
