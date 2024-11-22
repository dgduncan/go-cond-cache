package gocondcache

import (
	"errors"
	"net/http"
	"sync"
)

type conditionalHeader string

const (
	cacheControlMaxAge = "max-age"

	headerCacheControl = "cache-control"
	headerETAG         = "etag"

	headerIfMatch     = "If-Match"
	headerIfNoneMatch = "If-None-Match"

	headerLastModified      = "Last-Modified"
	headerIfMofifiedSince   = "If-Modified-Since"
	headerIfUnmodifiedSince = "If-Unmodified-Since"
)

type CacheTransport struct {
	Wrapped http.RoundTripper

	cache Cache
	lock  sync.RWMutex
}

func (c *CacheTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	ctx := r.Context()

	// check if request contains conditional header, exit early if not present
	etag := getConditionHeader(r)
	if etag == "" {
		resp, transportErr := c.Wrapped.RoundTrip(r)
		if transportErr != nil {
			return resp, transportErr
		}

		incomingETAG := resp.Header.Get(headerETAG)
		if incomingETAG == "" {

			return resp, transportErr
		}

		v, err := c.cache.Get(ctx, r.URL.Path)
		if err != nil {
			return resp, errors.Join(transportErr, err)
		}

		if v == nil {
			c.cache.Set(ctx, r.URL.Path, &CacheItem{
				ETAG: incomingETAG,
			})
		}
	}

	return nil, nil
}

func getConditionHeader(r *http.Request) string {
	etag := r.Header.Get(headerIfNoneMatch)
	if etag != "" {
		return etag
	}

	return r.Header.Get(headerIfMatch)
}
