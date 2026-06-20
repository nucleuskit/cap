package transport

type Options struct {
	Network    string
	Address    string
	ServerName string
	Timeout    TimeoutConfig
	TLS        TLSConfig
	Proxy      ProxyConfig
	Metadata   Metadata
	Hooks      []DialHook
}

type Option func(*Options)

func WithNetwork(network string) Option {
	return func(options *Options) {
		options.Network = network
	}
}

func WithAddress(address string) Option {
	return func(options *Options) {
		options.Address = address
	}
}

func WithServerName(serverName string) Option {
	return func(options *Options) {
		options.ServerName = serverName
	}
}

func WithTimeouts(timeout TimeoutConfig) Option {
	return func(options *Options) {
		options.Timeout = timeout
	}
}

func WithTLS(tls TLSConfig) Option {
	return func(options *Options) {
		options.TLS = tls.Clone()
	}
}

func WithProxy(proxy ProxyConfig) Option {
	return func(options *Options) {
		options.Proxy = proxy.Clone()
	}
}

func WithMetadata(key, value string) Option {
	return func(options *Options) {
		if options.Metadata == nil {
			options.Metadata = Metadata{}
		}
		options.Metadata[key] = value
	}
}

func WithMetadataMap(metadata map[string]string) Option {
	return func(options *Options) {
		options.Metadata = MergeMetadata(options.Metadata, metadata)
	}
}

func WithDialHooks(hooks ...DialHook) Option {
	return func(options *Options) {
		options.Hooks = append(options.Hooks, hooks...)
	}
}

func NewOptions(options ...Option) Options {
	values := Options{Network: string(NetworkTCP)}
	for _, option := range options {
		option(&values)
	}
	values.Metadata = CloneMetadata(values.Metadata)
	values.TLS = values.TLS.Clone()
	values.Proxy = values.Proxy.Clone()
	values.Hooks = append([]DialHook(nil), values.Hooks...)
	return values
}

func NewConfig(options ...Option) Config {
	values := NewOptions(options...)
	return Config{
		Network:    values.Network,
		Address:    values.Address,
		ServerName: values.ServerName,
		Timeout:    values.Timeout,
		TLS:        values.TLS.Clone(),
		Proxy:      values.Proxy.Clone(),
		Metadata:   CloneMetadata(values.Metadata),
		Hooks:      append([]DialHook(nil), values.Hooks...),
	}
}
