package metric

import (
	"context"
	"testing"
)

func TestNoopMeterImplementsMetricProtocol(t *testing.T) {
	var meter Meter = NewNoop()

	counter := meter.Counter("requests_total")
	counter.Add(context.Background(), 1, String("route", "/healthz"))
	counter.Record(context.Background(), 1)

	gauge := meter.Gauge("queue_depth")
	gauge.Set(context.Background(), 2, Any("queue", "default"))
	gauge.Record(context.Background(), 2)

	histogram := meter.Histogram("request_duration_ms", WithLabels("route"))
	histogram.Observe(context.Background(), 12.5, String("route", "/healthz"))
	histogram.Record(context.Background(), 13.5, String("route", "/readyz"))

	if len(meter.Snapshot()) != 0 {
		t.Fatalf("expected empty noop snapshot, got %#v", meter.Snapshot())
	}
}

func TestDescriptorCapturesKindLabelsAndBuckets(t *testing.T) {
	meter := NewNoop()
	histogram := meter.Histogram(
		"http.server.duration_ms",
		WithUnit("ms"),
		WithDescription("server latency"),
		WithLabels("method", "route"),
		WithBuckets(50, 100, 250),
	)

	descriptor := histogram.Descriptor()
	if descriptor.Kind != KindHistogram {
		t.Fatalf("kind = %q", descriptor.Kind)
	}
	if descriptor.Unit != "ms" || descriptor.Description != "server latency" {
		t.Fatalf("unexpected descriptor: %#v", descriptor)
	}
	if len(descriptor.Labels) != 2 || descriptor.Labels[0] != "method" || descriptor.Labels[1] != "route" {
		t.Fatalf("labels = %#v", descriptor.Labels)
	}
	if len(descriptor.Buckets) != 3 || descriptor.Buckets[1] != 100 {
		t.Fatalf("buckets = %#v", descriptor.Buckets)
	}

	descriptor.Buckets[1] = 1
	if histogram.Descriptor().Buckets[1] != 100 {
		t.Fatal("descriptor buckets should be immutable from caller mutations")
	}
}

func TestLabelsForDescriptorFiltersAndStabilizesValues(t *testing.T) {
	descriptor := Descriptor{Labels: []string{"method", "route"}}
	labels := LabelsFor(descriptor,
		String("route", "/readyz"),
		String("method", "GET"),
		Any("ignored", "value"),
	)

	if labels["method"] != "GET" || labels["route"] != "/readyz" {
		t.Fatalf("labels = %#v", labels)
	}
	if _, ok := labels["ignored"]; ok {
		t.Fatalf("unexpected undeclared label: %#v", labels)
	}
	if got := labels.Key(); got != `method="GET",route="/readyz"` {
		t.Fatalf("label key = %q", got)
	}
}

func TestMetricOptions(t *testing.T) {
	options := NewOptions(WithService("demo"), WithNamespace("http"))
	if options.Service != "demo" {
		t.Fatalf("expected service demo, got %q", options.Service)
	}
	if options.Namespace != "http" {
		t.Fatalf("expected namespace http, got %q", options.Namespace)
	}
	instrumentOptions := NewInstrumentOptions(WithUnit("ms"), WithDescription("latency"))
	if instrumentOptions.Unit != "ms" {
		t.Fatalf("expected unit ms, got %q", instrumentOptions.Unit)
	}
	if instrumentOptions.Description != "latency" {
		t.Fatalf("expected description latency, got %q", instrumentOptions.Description)
	}
	instrumentOptions = NewInstrumentOptions(WithBuckets(10, 20))
	if len(instrumentOptions.Buckets) != 2 || instrumentOptions.Buckets[1] != 20 {
		t.Fatalf("expected buckets, got %#v", instrumentOptions.Buckets)
	}
}
