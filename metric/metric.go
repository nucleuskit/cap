package metric

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

type InstrumentKind string

const (
	KindCounter   InstrumentKind = "counter"
	KindGauge     InstrumentKind = "gauge"
	KindHistogram InstrumentKind = "histogram"
)

type Attribute struct {
	Key   string
	Value any
}

type Descriptor struct {
	Kind        InstrumentKind
	Name        string
	Unit        string
	Description string
	Labels      []string
	Buckets     []float64
}

type LabelSet map[string]string

type Bucket struct {
	UpperBound float64
	Count      uint64
}

type Series struct {
	Descriptor Descriptor
	Labels     LabelSet
	Value      float64
	Count      uint64
	Sum        float64
	Buckets    []Bucket
}

type Instrument interface {
	Descriptor() Descriptor
	Record(ctx context.Context, value float64, attributes ...Attribute)
}

type Counter interface {
	Instrument
	Add(ctx context.Context, value float64, attributes ...Attribute)
}

type Gauge interface {
	Instrument
	Set(ctx context.Context, value float64, attributes ...Attribute)
}

type Histogram interface {
	Instrument
	Observe(ctx context.Context, value float64, attributes ...Attribute)
}

type Meter interface {
	Counter(name string, options ...InstrumentOption) Counter
	Gauge(name string, options ...InstrumentOption) Gauge
	Histogram(name string, options ...InstrumentOption) Histogram
	Snapshot() map[string]float64
}

type Snapshotter interface {
	SnapshotSeries() []Series
}

func String(key string, value string) Attribute {
	return Attribute{Key: key, Value: value}
}

func Int(key string, value int) Attribute {
	return Attribute{Key: key, Value: value}
}

func Bool(key string, value bool) Attribute {
	return Attribute{Key: key, Value: value}
}

func Any(key string, value any) Attribute {
	return Attribute{Key: key, Value: value}
}

func LabelsFor(descriptor Descriptor, attributes ...Attribute) LabelSet {
	if len(descriptor.Labels) == 0 {
		return nil
	}
	values := map[string]string{}
	declared := map[string]struct{}{}
	for _, label := range descriptor.Labels {
		declared[label] = struct{}{}
	}
	for _, attribute := range attributes {
		if strings.TrimSpace(attribute.Key) == "" {
			continue
		}
		if _, ok := declared[attribute.Key]; !ok {
			continue
		}
		values[attribute.Key] = fmt.Sprint(attribute.Value)
	}
	if len(values) == 0 {
		return nil
	}
	return LabelSet(values)
}

func (l LabelSet) Key() string {
	if len(l) == 0 {
		return ""
	}
	keys := make([]string, 0, len(l))
	for key := range l {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+fmt.Sprintf("%q", l[key]))
	}
	return strings.Join(parts, ",")
}

func (l LabelSet) Clone() LabelSet {
	if len(l) == 0 {
		return nil
	}
	values := make(LabelSet, len(l))
	for key, value := range l {
		values[key] = value
	}
	return values
}

func (d Descriptor) Clone() Descriptor {
	d.Labels = append([]string(nil), d.Labels...)
	d.Buckets = append([]float64(nil), d.Buckets...)
	return d
}

func (s Series) Clone() Series {
	s.Descriptor = s.Descriptor.Clone()
	s.Labels = s.Labels.Clone()
	s.Buckets = append([]Bucket(nil), s.Buckets...)
	return s
}
