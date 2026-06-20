package config

import (
	"context"
	"os"
	"sort"
	"strings"
)

type Values map[string]any

type Update struct {
	Values   Values
	Source   string
	Revision string
}

type Source struct {
	Name     string
	Kind     string
	Location string
	Priority int
}

type SourceValues struct {
	Source Source
	Values Values
}

type Loader interface {
	Load(context.Context) (Values, error)
}

type Watcher interface {
	Watch(context.Context) (<-chan Update, error)
}

type Scanner interface {
	Scan(ctx context.Context, target any) error
}

type Provider interface {
	Loader
	Watcher
	Scanner
	Sources() []Source
}

func CloneValues(values Values) Values {
	if values == nil {
		return Values{}
	}
	copied := make(Values, len(values))
	for key, value := range values {
		copied[key] = cloneValue(value)
	}
	return copied
}

func cloneValue(value any) any {
	switch typed := value.(type) {
	case Values:
		return CloneValues(typed)
	case map[string]any:
		return CloneValues(Values(typed))
	case []any:
		copied := make([]any, len(typed))
		for i, item := range typed {
			copied[i] = cloneValue(item)
		}
		return copied
	default:
		return value
	}
}

func SortSources(sources []Source) []Source {
	ordered := append([]Source(nil), sources...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return sourceLess(ordered[i], ordered[j])
	})
	return ordered
}

func MergeValues(values ...Values) Values {
	merged := Values{}
	for _, item := range values {
		mergeInto(merged, item)
	}
	return merged
}

func MergeSourceValues(values ...SourceValues) Values {
	ordered := append([]SourceValues(nil), values...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return sourceLess(ordered[i].Source, ordered[j].Source)
	})
	merged := Values{}
	for _, item := range ordered {
		mergeInto(merged, item.Values)
	}
	return merged
}

func ResolveEnv(values Values, lookup func(string) (string, bool)) Values {
	if lookup == nil {
		lookup = os.LookupEnv
	}
	resolved := CloneValues(values)
	resolveEnvValue(resolved, lookup)
	return resolved
}

func mergeInto(target Values, source Values) {
	for key, value := range source {
		if key == "" {
			continue
		}
		existing, exists := target[key]
		if exists {
			existingMap, existingOK := asValues(existing)
			incomingMap, incomingOK := asValues(value)
			if existingOK && incomingOK {
				merged := CloneValues(existingMap)
				mergeInto(merged, incomingMap)
				target[key] = merged
				continue
			}
		}
		target[key] = cloneValue(value)
	}
}

func resolveEnvValue(value any, lookup func(string) (string, bool)) any {
	switch typed := value.(type) {
	case Values:
		for key, item := range typed {
			typed[key] = resolveEnvValue(item, lookup)
		}
		return typed
	case map[string]any:
		for key, item := range typed {
			typed[key] = resolveEnvValue(item, lookup)
		}
		return typed
	case []any:
		for i, item := range typed {
			typed[i] = resolveEnvValue(item, lookup)
		}
		return typed
	case string:
		return expandEnvString(typed, lookup)
	default:
		return value
	}
}

func expandEnvString(value string, lookup func(string) (string, bool)) string {
	return os.Expand(value, func(key string) string {
		name, fallback, hasFallback := strings.Cut(key, ":-")
		if resolved, ok := lookup(name); ok {
			return resolved
		}
		if hasFallback {
			return fallback
		}
		return ""
	})
}

func asValues(value any) (Values, bool) {
	switch typed := value.(type) {
	case Values:
		return typed, true
	case map[string]any:
		return Values(typed), true
	default:
		return nil, false
	}
}

func sourceLess(left Source, right Source) bool {
	if left.Priority != right.Priority {
		return left.Priority < right.Priority
	}
	if left.Name != right.Name {
		return left.Name < right.Name
	}
	if left.Kind != right.Kind {
		return left.Kind < right.Kind
	}
	return left.Location < right.Location
}
