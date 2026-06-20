package log

import (
	"context"
	"sort"
	"time"
)

type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

const (
	FieldService    = "service"
	FieldComponent  = "component"
	FieldTraceID    = "trace_id"
	FieldSpanID     = "span_id"
	FieldRequestID  = "request_id"
	FieldError      = "error"
	FieldDuration   = "duration_ms"
	FieldStatusCode = "status_code"
	FieldRoute      = "http.route"
	FieldMethod     = "http.method"
)

type contextLoggerKey struct{}
type contextFieldsKey struct{}

type Field struct {
	Key   string
	Value any
	Type  string
}

type Entry struct {
	Level   Level
	Message string
	Fields  []Field
}

type Patch func(context.Context, Entry) Entry

type Logger interface {
	Debug(ctx context.Context, message string, fields ...Field)
	Info(ctx context.Context, message string, fields ...Field)
	Warn(ctx context.Context, message string, fields ...Field)
	Error(ctx context.Context, message string, fields ...Field)
}

func String(key string, value string) Field {
	return Field{Key: key, Value: value, Type: "string"}
}

func Int(key string, value int) Field {
	return Field{Key: key, Value: value, Type: "int"}
}

func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value, Type: "bool"}
}

func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value, Type: "float64"}
}

func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value, Type: "duration"}
}

func Error(err error) Field {
	return Field{Key: "error", Value: err, Type: "error"}
}

func Any(key string, value any) Field {
	return Field{Key: key, Value: value}
}

func Fields(values map[string]any) []Field {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fields := make([]Field, 0, len(values))
	for _, key := range keys {
		value := values[key]
		fields = append(fields, Any(key, value))
	}
	return fields
}

func WithLogger(ctx context.Context, logger Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		return ctx
	}
	return context.WithValue(ctx, contextLoggerKey{}, logger)
}

func FromContext(ctx context.Context) (Logger, bool) {
	if ctx == nil {
		return nil, false
	}
	logger, ok := ctx.Value(contextLoggerKey{}).(Logger)
	return logger, ok
}

func LoggerFromContext(ctx context.Context, fallback Logger) Logger {
	if logger, ok := FromContext(ctx); ok {
		return logger
	}
	return fallback
}

func WithContextFields(ctx context.Context, fields ...Field) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	values := ContextFields(ctx)
	values = append(values, fields...)
	return context.WithValue(ctx, contextFieldsKey{}, cloneFields(values))
}

func ContextFields(ctx context.Context) []Field {
	if ctx == nil {
		return nil
	}
	fields, _ := ctx.Value(contextFieldsKey{}).([]Field)
	return cloneFields(fields)
}

func NewEntry(ctx context.Context, level Level, message string, baseFields []Field, fields []Field, patches ...Patch) Entry {
	entry := Entry{
		Level:   level,
		Message: message,
		Fields:  make([]Field, 0, len(baseFields)+len(fields)+len(ContextFields(ctx))),
	}
	entry.Fields = append(entry.Fields, baseFields...)
	entry.Fields = append(entry.Fields, ContextFields(ctx)...)
	entry.Fields = append(entry.Fields, fields...)
	return ApplyPatches(ctx, entry, patches...)
}

func ApplyPatches(ctx context.Context, entry Entry, patches ...Patch) Entry {
	current := cloneEntry(entry)
	for _, patch := range patches {
		if patch == nil {
			continue
		}
		current = cloneEntry(patch(ctx, cloneEntry(current)))
	}
	return current
}

func WarnIf(ctx context.Context, logger Logger, err error, message string, fields ...Field) {
	if err == nil || logger == nil {
		return
	}
	logger.Warn(ctx, message, append(fields, Error(err))...)
}

func ErrorIf(ctx context.Context, logger Logger, err error, message string, fields ...Field) {
	if err == nil || logger == nil {
		return
	}
	logger.Error(ctx, message, append(fields, Error(err))...)
}

func cloneEntry(entry Entry) Entry {
	entry.Fields = cloneFields(entry.Fields)
	return entry
}

func cloneFields(fields []Field) []Field {
	if len(fields) == 0 {
		return nil
	}
	values := make([]Field, len(fields))
	copy(values, fields)
	return values
}
