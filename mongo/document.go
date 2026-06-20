package mongo

import (
	"context"
	"errors"
	"reflect"
	"time"
)

var (
	ErrInvalidCollection = errors.New("document collection is required")
	ErrInvalidID         = errors.New("document id is required")
	ErrNotFound          = errors.New("document not found")
	ErrConflict          = errors.New("document version conflict")
)

type Document struct {
	ID        string
	Fields    map[string]any
	Version   int64
	ExpiresAt int64
	Metadata  map[string]string
}

type Filter map[string]any

type Query struct {
	Collection string
	Filter     Filter
	Limit      int
	Offset     int
}

type Patch struct {
	Set   map[string]any
	Unset []string
}

type Store interface {
	Insert(ctx context.Context, collection string, doc Document, options ...WriteOption) (Document, error)
	Get(ctx context.Context, collection string, id string) (Document, error)
	Find(ctx context.Context, query Query) ([]Document, error)
	Replace(ctx context.Context, collection string, doc Document, options ...WriteOption) (Document, error)
	Update(ctx context.Context, collection string, id string, patch Patch, options ...WriteOption) (Document, error)
	Delete(ctx context.Context, collection string, id string) error
}

func NewDocument(id string, fields map[string]any, ttl time.Duration) Document {
	doc := Document{ID: id, Fields: cloneFields(fields)}
	if ttl > 0 {
		doc.ExpiresAt = time.Now().Add(ttl).UnixNano()
	}
	return doc
}

func (d Document) Expired(now time.Time) bool {
	return d.ExpiresAt > 0 && now.UnixNano() >= d.ExpiresAt
}

func (d Document) Clone() Document {
	clone := d
	clone.Fields = cloneFields(d.Fields)
	if d.Metadata != nil {
		clone.Metadata = make(map[string]string, len(d.Metadata))
		for key, value := range d.Metadata {
			clone.Metadata[key] = value
		}
	}
	return clone
}

func (p Patch) Clone() Patch {
	clone := Patch{
		Set:   cloneFields(p.Set),
		Unset: append([]string(nil), p.Unset...),
	}
	return clone
}

func Match(doc Document, filter Filter) bool {
	for key, want := range filter {
		var got any
		switch key {
		case "_id", "id":
			got = doc.ID
		case "_version", "version":
			got = doc.Version
		default:
			got = doc.Fields[key]
		}
		if !reflect.DeepEqual(got, want) {
			return false
		}
	}
	return true
}

func cloneFields(fields map[string]any) map[string]any {
	if fields == nil {
		return nil
	}
	clone := make(map[string]any, len(fields))
	for key, value := range fields {
		clone[key] = cloneValue(value)
	}
	return clone
}

func cloneValue(value any) any {
	switch typed := value.(type) {
	case []byte:
		return append([]byte(nil), typed...)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = cloneValue(item)
		}
		return out
	case map[string]any:
		return cloneFields(typed)
	case map[string]string:
		out := make(map[string]string, len(typed))
		for key, item := range typed {
			out[key] = item
		}
		return out
	default:
		return value
	}
}
