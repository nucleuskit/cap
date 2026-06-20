package redis

type Options struct {
	Address   string
	Addrs     []string
	Database  int
	Namespace string
	Cluster   ClusterConfig
	Pool      PoolConfig
	Retry     RetryConfig
	Timeout   TimeoutConfig
	TLS       TLSConfig
	Hooks     []OperationHook
}

type Option func(*Options)

func WithAddress(address string) Option {
	return func(options *Options) {
		options.Address = address
	}
}

func WithNamespace(namespace string) Option {
	return func(options *Options) {
		options.Namespace = namespace
	}
}

func WithDatabase(database int) Option {
	return func(options *Options) {
		options.Database = database
	}
}

func WithCluster(addrs ...string) Option {
	return func(options *Options) {
		options.Addrs = append(options.Addrs, addrs...)
		options.Cluster.Enabled = true
		options.Cluster.Addrs = append(options.Cluster.Addrs, addrs...)
	}
}

func WithPool(pool PoolConfig) Option {
	return func(options *Options) {
		options.Pool = pool
	}
}

func WithRetry(retry RetryConfig) Option {
	return func(options *Options) {
		options.Retry = retry
	}
}

func WithTimeouts(timeout TimeoutConfig) Option {
	return func(options *Options) {
		options.Timeout = timeout
	}
}

func WithTLS(tls TLSConfig) Option {
	return func(options *Options) {
		options.TLS = tls
	}
}

func WithOperationHooks(hooks ...OperationHook) Option {
	return func(options *Options) {
		options.Hooks = append(options.Hooks, hooks...)
	}
}

func NewOptions(options ...Option) Options {
	values := Options{}
	for _, option := range options {
		option(&values)
	}
	values.Addrs = append([]string(nil), values.Addrs...)
	values.Cluster.Addrs = append([]string(nil), values.Cluster.Addrs...)
	values.Hooks = append([]OperationHook(nil), values.Hooks...)
	return values
}
