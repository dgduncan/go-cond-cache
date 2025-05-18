package caches

import (
	"errors"
	"fmt"
)

var (
	// ErrCacheItemExpired is returned when a cache item is expired and needs to be revalidated
	ErrCacheItemExpired = errors.New("cache item expired")
	// ErrNoCacheItem is returned when the key is not found in the cache
	ErrNoCacheItem = errors.New("no value found in cache")
)

// ValidationError represents an validation error on the initial creation of a cache.
type ValidationError struct {
	Reason string
}

// Error returns the string value of the error
func (ve ValidationError) Error() string {
	return fmt.Sprintf("creation of cache failed for reason : %s ", ve.Reason)
}
