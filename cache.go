package gocondcache

import (
	"context"
	"time"
)

type CacheItem struct {
	ETAG       string
	Response   []byte
	Expiration time.Time // NOTE : look to see if expiration needs to be set from cache-control headers. This represents
}

type Cache interface {
	Get(ctx context.Context, k string) (*CacheItem, error)
	Set(ctx context.Context, k string, v *CacheItem) error
	Update(ctx context.Context, k string, expiration time.Time) error
}
