package trace

import (
	"context"
	"testing"
)

func TestNoopTracerImplementsTracer(t *testing.T) {
	var _ Tracer = NewNoop()

	ctx := context.Background()
	tracer := NewNoop()
	spanCtx, span := tracer.Start(ctx, "operation")
	if spanCtx != ctx {
		t.Fatal("noop tracer should return the original context")
	}
	span.SetAttribute("component", "test")
	span.RecordError(nil)
	span.End()
}

func TestTraceParentAndBaggageRoundTrip(t *testing.T) {
	spanContext := SpanContext{
		TraceID:    "4bf92f3577b34da6a3ce929d0e0e4736",
		SpanID:     "00f067aa0ba902b7",
		TraceFlags: "01",
	}
	ctx := ContextWithSpanContext(context.Background(), spanContext)
	ctx = WithBaggageItem(ctx, "tenant", "acme")
	ctx = WithBaggageItem(ctx, "region", "us")

	carrier := Carrier{}
	InjectContext(ctx, carrier)
	if got := carrier.Get(HeaderTraceParent); got != "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01" {
		t.Fatalf("traceparent = %q", got)
	}
	if got := carrier.Get(HeaderBaggage); got != "region=us,tenant=acme" {
		t.Fatalf("baggage = %q", got)
	}

	extracted := ExtractContext(context.Background(), carrier)
	got, ok := SpanContextFromContext(extracted)
	if !ok {
		t.Fatal("expected extracted span context")
	}
	if got.TraceID != spanContext.TraceID || got.SpanID != spanContext.SpanID || !got.Remote {
		t.Fatalf("unexpected extracted span context: %#v", got)
	}
	if baggage := BaggageFromContext(extracted); baggage["tenant"] != "acme" || baggage["region"] != "us" {
		t.Fatalf("unexpected baggage: %#v", baggage)
	}
}

func TestNoopTracerPropagatesW3CContext(t *testing.T) {
	tracer := NewNoop()
	ctx := tracer.Extract(context.Background(), Carrier{
		HeaderTraceParent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		HeaderBaggage:     "tenant=acme",
	})

	carrier := Carrier{}
	tracer.Inject(ctx, carrier)
	if got := carrier.Get(HeaderTraceParent); got != "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01" {
		t.Fatalf("traceparent = %q", got)
	}
	if got := carrier.Get(HeaderBaggage); got != "tenant=acme" {
		t.Fatalf("baggage = %q", got)
	}
}

func TestTraceOptions(t *testing.T) {
	options := NewOptions(WithService("orders"), WithSampler("always_on"))
	if options.Service != "orders" {
		t.Fatalf("expected service orders, got %q", options.Service)
	}
	if options.Sampler != "always_on" {
		t.Fatalf("expected sampler always_on, got %q", options.Sampler)
	}
}
