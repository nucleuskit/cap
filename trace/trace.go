package trace

import (
	"context"
	"net/url"
	"sort"
	"strings"
)

const (
	HeaderTraceParent = "traceparent"
	HeaderBaggage     = "baggage"
)

type spanContextKey struct{}
type baggageKey struct{}

type Attribute struct {
	Key   string
	Value any
}

type SpanContext struct {
	TraceID    string
	SpanID     string
	TraceFlags string
	Remote     bool
}

type Carrier map[string]string
type Baggage map[string]string

type Span interface {
	Context() SpanContext
	SetAttribute(key string, value any)
	RecordError(err error)
	End()
}

type Tracer interface {
	Start(ctx context.Context, name string, attributes ...Attribute) (context.Context, Span)
	Inject(ctx context.Context, carrier Carrier)
	Extract(ctx context.Context, carrier Carrier) context.Context
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

func (c Carrier) Get(key string) string {
	if c == nil {
		return ""
	}
	if value, ok := c[key]; ok {
		return value
	}
	for candidate, value := range c {
		if strings.EqualFold(candidate, key) {
			return value
		}
	}
	return ""
}

func (c Carrier) Set(key string, value string) {
	if c == nil || strings.TrimSpace(key) == "" || value == "" {
		return
	}
	c[strings.ToLower(key)] = value
}

func ContextWithSpanContext(ctx context.Context, spanContext SpanContext) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, spanContextKey{}, spanContext)
}

func SpanContextFromContext(ctx context.Context) (SpanContext, bool) {
	if ctx == nil {
		return SpanContext{}, false
	}
	spanContext, ok := ctx.Value(spanContextKey{}).(SpanContext)
	return spanContext, ok
}

func WithBaggage(ctx context.Context, baggage Baggage) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, baggageKey{}, cloneBaggage(baggage))
}

func WithBaggageItem(ctx context.Context, key string, value string) context.Context {
	baggage := BaggageFromContext(ctx)
	if baggage == nil {
		baggage = Baggage{}
	}
	baggage[key] = value
	return WithBaggage(ctx, baggage)
}

func BaggageFromContext(ctx context.Context) Baggage {
	if ctx == nil {
		return nil
	}
	baggage, _ := ctx.Value(baggageKey{}).(Baggage)
	return cloneBaggage(baggage)
}

func InjectContext(ctx context.Context, carrier Carrier) {
	if carrier == nil {
		return
	}
	if spanContext, ok := SpanContextFromContext(ctx); ok {
		carrier.Set(HeaderTraceParent, FormatTraceParent(spanContext))
	}
	if baggage := FormatBaggage(BaggageFromContext(ctx)); baggage != "" {
		carrier.Set(HeaderBaggage, baggage)
	}
}

func ExtractContext(ctx context.Context, carrier Carrier) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if spanContext, ok := ParseTraceParent(carrier.Get(HeaderTraceParent)); ok {
		spanContext.Remote = true
		ctx = ContextWithSpanContext(ctx, spanContext)
	}
	if baggage := ParseBaggage(carrier.Get(HeaderBaggage)); len(baggage) > 0 {
		ctx = WithBaggage(ctx, baggage)
	}
	return ctx
}

func FormatTraceParent(spanContext SpanContext) string {
	if spanContext.TraceFlags == "" {
		spanContext.TraceFlags = "00"
	}
	return "00-" + spanContext.TraceID + "-" + spanContext.SpanID + "-" + spanContext.TraceFlags
}

func ParseTraceParent(value string) (SpanContext, bool) {
	parts := strings.Split(strings.TrimSpace(value), "-")
	if len(parts) != 4 || parts[0] != "00" {
		return SpanContext{}, false
	}
	traceID, spanID, flags := strings.ToLower(parts[1]), strings.ToLower(parts[2]), strings.ToLower(parts[3])
	if !validHex(traceID, 32) || !validHex(spanID, 16) || !validHex(flags, 2) {
		return SpanContext{}, false
	}
	return SpanContext{TraceID: traceID, SpanID: spanID, TraceFlags: flags}, true
}

func FormatBaggage(baggage Baggage) string {
	if len(baggage) == 0 {
		return ""
	}
	keys := make([]string, 0, len(baggage))
	for key := range baggage {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]string, 0, len(keys))
	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			continue
		}
		items = append(items, url.QueryEscape(key)+"="+url.QueryEscape(baggage[key]))
	}
	return strings.Join(items, ",")
}

func ParseBaggage(value string) Baggage {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	baggage := Baggage{}
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		item, _, _ = strings.Cut(item, ";")
		key, rawValue, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		decodedKey, err := url.QueryUnescape(key)
		if err == nil {
			key = decodedKey
		}
		decodedValue, err := url.QueryUnescape(strings.TrimSpace(rawValue))
		if err != nil {
			decodedValue = strings.TrimSpace(rawValue)
		}
		baggage[key] = decodedValue
	}
	return baggage
}

func cloneBaggage(baggage Baggage) Baggage {
	if len(baggage) == 0 {
		return nil
	}
	values := make(Baggage, len(baggage))
	for key, value := range baggage {
		values[key] = value
	}
	return values
}

func validHex(value string, length int) bool {
	if len(value) != length || strings.Count(value, "0") == length {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}
