package local

import (
	"context"
	"sync"

	gocondcache "github.com/dgduncan/go-cond-cache"
)

type BasicCache struct {
	cache map[string]*gocondcache.CacheItem

	lock sync.RWMutex
}

func (bc *BasicCache) Get(_ context.Context, key string) (*gocondcache.CacheItem, error) {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	val, found := bc.cache[key]
	if !found {
		return nil, gocondcache.ErrNotFound
	}

	return val, nil
}

func (bc *BasicCache) Set(_ context.Context, key string, item *gocondcache.CacheItem) error {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	bc.cache[key] = item

	return nil
}

func NewBasicCache() BasicCache {
	return BasicCache{
		cache: make(map[string]*gocondcache.CacheItem),
		lock:  sync.RWMutex{},
	}
}
