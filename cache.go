package gocondcache

import (
	"context"
	"time"
)

// CacheItem represents a cached HTTP response with its associated metadata.
// It contains the ETag for conditional request validation, the response body,
// and an expiration time for cache invalidation.
type CacheItem struct {
	ETAG       string
	Response   []byte
	Expiration time.Time
}

// Cache defines the interface for cache operations across different storage implementations.
// It provides methods for getting, setting, and updating cache items using a consistent API.
// Here, k represents a generic `key` for use in the cache.
// The implementation of Update in some caches will can be modeled as having to make multiples calls
// to the cache in the form of Get and then a Set due to the lack of a first class `Update or Upsert`. This interface may change in the future in order to allow
// for more straightforward implementations in these cases.
type Cache interface {
	Get(ctx context.Context, k string) (*CacheItem, error)
	Set(ctx context.Context, k string, v *CacheItem) error
	Update(ctx context.Context, k string, expiration time.Time) error
}
