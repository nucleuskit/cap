package trace

import "context"

type noopTracer struct{}

type noopSpan struct{}

func NewNoop() Tracer {
	return noopTracer{}
}

func (noopTracer) Start(ctx context.Context, _ string, _ ...Attribute) (context.Context, Span) {
	return ctx, noopSpan{}
}

func (noopTracer) Inject(ctx context.Context, carrier Carrier) {
	InjectContext(ctx, carrier)
}

func (noopTracer) Extract(ctx context.Context, carrier Carrier) context.Context {
	return ExtractContext(ctx, carrier)
}

func (noopSpan) Context() SpanContext {
	return SpanContext{}
}

func (noopSpan) SetAttribute(string, any) {}
func (noopSpan) RecordError(error)        {}
func (noopSpan) End()                     {}
