package store

import (
	"context"
	"time"
)

type Entry struct {
	Key       string
	Value     []byte
	ExpiresAt int64
	Metadata  map[string]string
}

type Loader func(ctx context.Context, key string) (Entry, error)

type Store interface {
	Get(ctx context.Context, key string) (Entry, error)
	Set(ctx context.Context, entry Entry) error
	Add(ctx context.Context, entry Entry) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]Entry, error)
}

type CacheAside interface {
	GetOrSet(ctx context.Context, key string, ttl time.Duration, load Loader) (Entry, error)
}

func NewEntry(key string, value []byte, ttl time.Duration) Entry {
	entry := Entry{Key: key, Value: append([]byte(nil), value...)}
	if ttl > 0 {
		entry.ExpiresAt = time.Now().Add(ttl).UnixNano()
	}
	return entry
}

func (e Entry) Expired(now time.Time) bool {
	return e.ExpiresAt > 0 && now.UnixNano() >= e.ExpiresAt
}

func (e Entry) Clone() Entry {
	clone := e
	clone.Value = append([]byte(nil), e.Value...)
	if e.Metadata != nil {
		clone.Metadata = make(map[string]string, len(e.Metadata))
		for key, value := range e.Metadata {
			clone.Metadata[key] = value
		}
	}
	return clone
}
