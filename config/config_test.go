package config

import (
	"context"
	"testing"
)

func TestNoopLoaderImplementsConfigInterfaces(t *testing.T) {
	var _ Loader = NewNoop()
	var _ Watcher = NewNoop()
	var _ Scanner = NewNoop()

	loader := NewNoop()
	values, err := loader.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(values) != 0 {
		t.Fatalf("expected empty config, got %#v", values)
	}
}

func TestConfigOptions(t *testing.T) {
	options := NewOptions(WithSource("file"), WithPath("configs/app.yaml"), WithPriority(20))
	if options.Source != "file" {
		t.Fatalf("expected source file, got %q", options.Source)
	}
	if options.Path != "configs/app.yaml" {
		t.Fatalf("expected path configs/app.yaml, got %q", options.Path)
	}
	if options.Priority != 20 {
		t.Fatalf("expected priority 20, got %d", options.Priority)
	}
}

func TestSortSourcesOrdersByPriorityWithoutMutatingInput(t *testing.T) {
	sources := []Source{
		{Name: "cache", Kind: "file", Location: "cache.yaml", Priority: 90},
		{Name: "remote", Kind: "remote", Location: "nacos", Priority: 10},
		{Name: "local", Kind: "file", Location: "app.yaml", Priority: 10},
	}

	ordered := SortSources(sources)

	if ordered[0].Name != "local" || ordered[1].Name != "remote" || ordered[2].Name != "cache" {
		t.Fatalf("unexpected order: %#v", ordered)
	}
	if sources[0].Name != "cache" {
		t.Fatalf("expected original slice to remain unchanged, got %#v", sources)
	}
}

func TestCloneValuesCopiesNestedMapsAndSlices(t *testing.T) {
	values := Values{
		"service": map[string]any{"name": "demo"},
		"ports":   []any{8080},
	}

	cloned := CloneValues(values)
	cloned["service"].(Values)["name"] = "changed"
	cloned["ports"].([]any)[0] = 9090

	if values["service"].(map[string]any)["name"] != "demo" {
		t.Fatalf("expected nested map to be cloned, got %#v", values)
	}
	if values["ports"].([]any)[0] != 8080 {
		t.Fatalf("expected nested slice to be cloned, got %#v", values)
	}
}

func TestMergeValuesDeepMergesWithoutMutatingInputs(t *testing.T) {
	base := Values{
		"service": Values{"name": "orders", "port": 8080},
		"debug":   false,
	}
	override := Values{
		"service": map[string]any{"port": 9090},
	}

	merged := MergeValues(base, override)
	merged["service"].(Values)["name"] = "changed"

	service := MergeValues(base, override)["service"].(Values)
	if service["name"] != "orders" || service["port"] != 9090 {
		t.Fatalf("unexpected merged service: %#v", service)
	}
	if base["service"].(Values)["name"] != "orders" {
		t.Fatalf("merge mutated base: %#v", base)
	}
}

func TestMergeSourceValuesOrdersByPriority(t *testing.T) {
	merged := MergeSourceValues(
		SourceValues{Source: Source{Name: "remote", Priority: 20}, Values: Values{"port": 8080}},
		SourceValues{Source: Source{Name: "local", Priority: 30}, Values: Values{"port": 9090}},
	)
	if merged["port"] != 9090 {
		t.Fatalf("expected higher priority source to override, got %#v", merged)
	}
}

func TestResolveEnvExpandsDefaultsAndNestedValues(t *testing.T) {
	values := Values{
		"dsn": "${DB_DSN:-memory}",
		"nested": []any{
			Values{"token": "${TOKEN}"},
		},
	}
	resolved := ResolveEnv(values, func(key string) (string, bool) {
		switch key {
		case "TOKEN":
			return "secret", true
		default:
			return "", false
		}
	})

	if resolved["dsn"] != "memory" {
		t.Fatalf("expected default dsn, got %#v", resolved["dsn"])
	}
	nested := resolved["nested"].([]any)[0].(Values)
	if nested["token"] != "secret" {
		t.Fatalf("expected token expansion, got %#v", nested)
	}
}
