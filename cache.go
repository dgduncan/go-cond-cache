package gocondcache

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("cache item not found")
)

type CacheItem struct {
	ETAG       string
	Response   []byte
	Expiration time.Time // NOTE : look to see if expiration needs to be set from cache-control headers. This represents
}

type Cache interface {
	Get(ctx context.Context, k string) (*CacheItem, error)
	Set(ctx context.Context, k string, v *CacheItem) error
}
