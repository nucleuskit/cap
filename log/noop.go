package log

import "context"

type noopLogger struct{}

func NewNoop() Logger {
	return noopLogger{}
}

func (noopLogger) Debug(context.Context, string, ...Field) {}
func (noopLogger) Info(context.Context, string, ...Field)  {}
func (noopLogger) Warn(context.Context, string, ...Field)  {}
func (noopLogger) Error(context.Context, string, ...Field) {}
