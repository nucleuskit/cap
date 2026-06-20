package metric

import "context"

type noopMeter struct{}

type noopInstrument struct {
	descriptor Descriptor
}

func NewNoop() Meter {
	return noopMeter{}
}

func (noopMeter) Counter(name string, options ...InstrumentOption) Counter {
	return noopInstrument{descriptor: descriptor(KindCounter, name, options...)}
}

func (noopMeter) Gauge(name string, options ...InstrumentOption) Gauge {
	return noopInstrument{descriptor: descriptor(KindGauge, name, options...)}
}

func (noopMeter) Histogram(name string, options ...InstrumentOption) Histogram {
	return noopInstrument{descriptor: descriptor(KindHistogram, name, options...)}
}

func (noopMeter) Snapshot() map[string]float64 {
	return map[string]float64{}
}

func (i noopInstrument) Descriptor() Descriptor {
	return i.descriptor.Clone()
}

func (noopInstrument) Record(context.Context, float64, ...Attribute) {}

func (noopInstrument) Add(context.Context, float64, ...Attribute) {}

func (noopInstrument) Set(context.Context, float64, ...Attribute) {}

func (noopInstrument) Observe(context.Context, float64, ...Attribute) {}

func descriptor(kind InstrumentKind, name string, options ...InstrumentOption) Descriptor {
	values := NewInstrumentOptions(options...)
	return Descriptor{
		Kind:        kind,
		Name:        name,
		Unit:        values.Unit,
		Description: values.Description,
		Labels:      append([]string(nil), values.Labels...),
		Buckets:     append([]float64(nil), values.Buckets...),
	}
}
