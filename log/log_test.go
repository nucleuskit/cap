package log

import (
	"bytes"
	"context"
	"testing"
	"time"
)

type recordingLogger struct {
	fields []Field
}

func (r *recordingLogger) Debug(context.Context, string, ...Field) {}
func (r *recordingLogger) Info(_ context.Context, _ string, fields ...Field) {
	r.fields = append([]Field(nil), fields...)
}
func (r *recordingLogger) Warn(context.Context, string, ...Field)  {}
func (r *recordingLogger) Error(context.Context, string, ...Field) {}

func TestNoopLoggerImplementsLogger(t *testing.T) {
	var logger Logger = NewNoop()
	logger.Info(context.Background(), "hello", String("component", "test"))
}

func TestContextLoggerAndFields(t *testing.T) {
	logger := &recordingLogger{}
	ctx := WithLogger(context.Background(), logger)
	ctx = WithContextFields(ctx, String(FieldService, "orders"), String(FieldTraceID, "trace-1"))

	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("expected logger in context")
	}
	if got != logger {
		t.Fatal("expected context logger to round-trip")
	}

	fields := ContextFields(ctx)
	fields[0] = String(FieldService, "mutated")
	if ContextFields(ctx)[0].Value == "mutated" {
		t.Fatal("context fields should be immutable from caller mutations")
	}

	fallback := NewNoop()
	if LoggerFromContext(context.Background(), fallback) != fallback {
		t.Fatal("expected fallback logger when context has none")
	}
}

func TestFieldHelpersUseStableTypes(t *testing.T) {
	fields := []Field{
		String(FieldTraceID, "trace-1"),
		Int(FieldStatusCode, 200),
		Bool("cache.hit", true),
		Float64("sample.rate", 0.5),
		Duration(FieldDuration, time.Second),
		Any("payload", map[string]any{"ok": true}),
	}

	wantTypes := []string{"string", "int", "bool", "float64", "duration", ""}
	for i, want := range wantTypes {
		if fields[i].Type != want {
			t.Fatalf("field %d type = %q, want %q", i, fields[i].Type, want)
		}
	}
}

func TestApplyPatchesCopiesEntryFields(t *testing.T) {
	entry := Entry{Level: LevelInfo, Message: "hello", Fields: []Field{String("component", "test")}}
	patched := ApplyPatches(context.Background(), entry, func(_ context.Context, entry Entry) Entry {
		entry.Fields = append(entry.Fields, String("patched", "true"))
		return entry
	})

	if len(patched.Fields) != 2 {
		t.Fatalf("expected patched fields, got %#v", patched.Fields)
	}
	if len(entry.Fields) != 1 {
		t.Fatalf("original entry fields should not be mutated, got %#v", entry.Fields)
	}

	var buf bytes.Buffer
	options := NewOptions(WithWriter(&buf), WithPatch(func(context.Context, Entry) Entry { return entry }))
	if options.Writer != &buf {
		t.Fatal("expected writer option to be retained")
	}
	if len(options.Patches) != 1 {
		t.Fatalf("expected one patch, got %d", len(options.Patches))
	}
}
