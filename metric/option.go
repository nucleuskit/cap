package metric

import "sort"

type Options struct {
	Service   string
	Namespace string
}

type Option func(*Options)

func WithService(service string) Option {
	return func(options *Options) {
		options.Service = service
	}
}

func WithNamespace(namespace string) Option {
	return func(options *Options) {
		options.Namespace = namespace
	}
}

func NewOptions(options ...Option) Options {
	values := Options{}
	for _, option := range options {
		option(&values)
	}
	return values
}

type InstrumentOptions struct {
	Unit        string
	Description string
	Labels      []string
	Buckets     []float64
}

type InstrumentOption func(*InstrumentOptions)

func WithUnit(unit string) InstrumentOption {
	return func(options *InstrumentOptions) {
		options.Unit = unit
	}
}

func WithDescription(description string) InstrumentOption {
	return func(options *InstrumentOptions) {
		options.Description = description
	}
}

func WithLabels(labels ...string) InstrumentOption {
	return func(options *InstrumentOptions) {
		options.Labels = append(options.Labels, labels...)
	}
}

func WithBuckets(buckets ...float64) InstrumentOption {
	return func(options *InstrumentOptions) {
		options.Buckets = append(options.Buckets, buckets...)
	}
}

func NewInstrumentOptions(options ...InstrumentOption) InstrumentOptions {
	values := InstrumentOptions{}
	for _, option := range options {
		option(&values)
	}
	values.Labels = append([]string(nil), values.Labels...)
	values.Buckets = append([]float64(nil), values.Buckets...)
	sort.Float64s(values.Buckets)
	return values
}
