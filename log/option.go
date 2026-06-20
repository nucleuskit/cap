package log

import "io"

type Options struct {
	Service    string
	Level      Level
	CallerSkip int
	Fields     []Field
	Writer     io.Writer
	Patches    []Patch
}

type Option func(*Options)

func WithService(service string) Option {
	return func(options *Options) {
		options.Service = service
	}
}

func WithLevel(level string) Option {
	return func(options *Options) {
		options.Level = Level(level)
	}
}

func WithCallerSkip(skip int) Option {
	return func(options *Options) {
		options.CallerSkip = skip
	}
}

func WithFields(fields ...Field) Option {
	return func(options *Options) {
		options.Fields = append(options.Fields, fields...)
	}
}

func WithWriter(writer io.Writer) Option {
	return func(options *Options) {
		options.Writer = writer
	}
}

func WithPatch(patch Patch) Option {
	return func(options *Options) {
		if patch != nil {
			options.Patches = append(options.Patches, patch)
		}
	}
}

func NewOptions(options ...Option) Options {
	values := Options{Level: LevelInfo}
	for _, option := range options {
		option(&values)
	}
	values.Fields = append([]Field(nil), values.Fields...)
	values.Patches = append([]Patch(nil), values.Patches...)
	return values
}
