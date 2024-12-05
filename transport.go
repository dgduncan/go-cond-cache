package gocondcache

import (
	"bufio"
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"
)

// type conditionalHeader string

const (
	cacheControlMaxAge = "max-age"

	headerCacheControl = "cache-control"
	headerETAG         = "etag"

	headerIfMatch     = "If-Match"
	headerIfNoneMatch = "If-None-Match"

	// headerLastModified      = "Last-Modified"
	// headerIfMofifiedSince   = "If-Modified-Since"
	// headerIfUnmodifiedSince = "If-Unmodified-Since"
)

type CacheTransport struct {
	Wrapped http.RoundTripper

	cache Cache
	now   func() time.Time
}

func (c *CacheTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	ctx := r.Context()

	// check if request contains conditional header, exit early if not present
	item, err := c.cache.Get(ctx, r.URL.String())
	if err == nil {
		// cached item is still valid
		if c.now().UTC().Before(item.Expiration) {
			nr := bufio.NewReader(bytes.NewReader(item.Response))
			return http.ReadResponse(nr, nil)
		}

		// item has been found in the cache but is expired
		// check if item is still valid by adding etag to conditional requesst
		r.Header.Add(headerIfNoneMatch, item.ETAG)
	}

	resp, transportError := c.Wrapped.RoundTrip(r)
	if resp.StatusCode == http.StatusNotModified {
		// cache item as been revalidated as the response is 304
		maxAge := getMaxAge(resp)

		if err := c.cache.Update(ctx, r.URL.String(), c.now().UTC().Add(maxAge)); err != nil {
			return resp, errors.Join(err, transportError) // return original http response and error
		}

		nr := bufio.NewReader(bytes.NewReader(item.Response))
		return http.ReadResponse(nr, nil)
	}

	// check if response contains conditional request header i.e etag
	if getETAGHeader(resp) == "" { // if no etag header is found, we don't cache the response
		return resp, transportError
	}

	maxAge := getMaxAge(resp)
	resBytes, _ := httputil.DumpResponse(resp, true)
	if err := c.cache.Set(ctx, r.URL.String(), &CacheItem{
		ETAG:       resp.Header.Get(headerETAG),
		Response:   resBytes,
		Expiration: c.now().UTC().Add(maxAge),
	}); err != nil {
		slog.Debug("error caching response", "error", err)
	}

	return resp, transportError
}

func getMaxAge(r *http.Response) time.Duration {
	// Get the Cache-Control header value
	cacheControl := getCacheControlHeader(r)
	if cacheControl == "" {
		return 0
	}

	// Split the header value by commas
	directives := strings.Split(cacheControl, ",")
	// Trim whitespace around each directive
	for i, directive := range directives {
		directives[i] = strings.TrimSpace(directive)
	}

	var maxAge time.Duration
	// Find the max-age directive
	for _, directive := range directives {
		if strings.HasPrefix(directive, cacheControlMaxAge) {
			// Split the directive by the equals sign
			parts := strings.Split(directive, "=")
			if parts[1] == "" {
				return 0
			}
			// The second part is the max-age value
			maxAge, _ = time.ParseDuration(parts[1] + "s")
			break
		}
	}

	return maxAge
}

func getETAGHeader(r *http.Response) string {
	return r.Header.Get(headerETAG)
}

func getCacheControlHeader(r *http.Response) string {
	return r.Header.Get(headerCacheControl)
}

// NewETAG placeholder
func New(cache Cache, now func() time.Time) func(http.RoundTripper) http.RoundTripper {
	nowFunc := now
	if now == nil {
		nowFunc = time.Now
	}

	return func(rt http.RoundTripper) http.RoundTripper {
		return &CacheTransport{Wrapped: rt, cache: cache, now: nowFunc}
	}
}
