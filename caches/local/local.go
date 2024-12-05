package local

import (
	"context"
	"sync"
	"time"

	gocondcache "github.com/dgduncan/go-cond-cache"
)

type BasicCache struct {
	cache map[string]*gocondcache.CacheItem

	lock *sync.RWMutex
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

func (bc *BasicCache) Update(_ context.Context, key string, expiration time.Time) error {
	ctx := context.TODO()

	// NOTE : this may cause a race condition because of the use of a double lock compared to a larger single lock
	// that encompasses the read and write operations
	item, err := bc.Get(ctx, key)
	if err != nil {
		return err
	}
	item.Expiration = expiration

	return bc.Set(ctx, key, item)
}

func NewBasicCache() BasicCache {
	return BasicCache{
		cache: make(map[string]*gocondcache.CacheItem),
		lock:  &sync.RWMutex{},
	}
}
