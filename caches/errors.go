package caches

import (
	"errors"
	"fmt"
)

type ValidationError struct {
	Reason string
}

func (ve ValidationError) Error() string {
	return fmt.Sprintf("creation of cache failed for reason : %s ", ve.Reason)

}

// TODO : This should be a more specific error
var (
	ErrCacheItemExpired = errors.New("cache item expired")
	ErrNoCacheItem      = errors.New("no value found in cache")
)
