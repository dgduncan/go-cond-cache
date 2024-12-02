package gocondcache

import (
	"bufio"
	"bytes"
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
}

func (c *CacheTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	ctx := r.Context()

	// check if request contains conditional header, exit early if not present
	item, err := c.cache.Get(ctx, r.URL.String())
	if err == nil {
		// cached item is still valid
		if time.Now().UTC().Before(item.Expiration) {
			nr := bufio.NewReader(bytes.NewReader(item.Response))
			return http.ReadResponse(nr, nil)
		}

		// check if item is still valid by adding etag to conditional requesst
		r.Header.Add(headerIfNoneMatch, item.ETAG)
	}

	resp, transportError := c.Wrapped.RoundTrip(r)
	if resp.StatusCode == http.StatusNotModified {
		item, err := c.cache.Get(ctx, r.URL.String())
		if err != nil {
			return resp, transportError
		}

		maxAge := getMaxAge(resp)
		item.Expiration = time.Now().Add(maxAge)

		if err := c.cache.Set(ctx, r.URL.String(), item); err != nil {
			return resp, transportError
		}

		nr := bufio.NewReader(bytes.NewReader(item.Response))
		return http.ReadResponse(nr, nil)
	}

	if getETAGHeader(resp) == "" { // if no etag header is found, we don't cache the response
		return resp, transportError
	}

	maxAge := getMaxAge(resp)
	resBytes, _ := httputil.DumpResponse(resp, true)
	if err := c.cache.Set(ctx, r.URL.String(), &CacheItem{
		ETAG:       resp.Header.Get(headerETAG),
		Response:   resBytes,
		Expiration: time.Now().Add(maxAge),
	}); err != nil {
		slog.Debug("error caching response", "error", err)
	}

	return resp, transportError
}

// func containsConditionalHeader(r *http.Request) bool {
// 	headers := r.Header

// 	for k, v := range headers {
// 		fmt.Println(k, v)
// 	}

// 	return false
// }

// func getConditionHeader(r *http.Request) string {
// 	etag := r.Header.Get(headerIfNoneMatch)
// 	if etag != "" {
// 		return etag
// 	}

// 	return r.Header.Get(headerIfMatch)
// }

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
func New(cache Cache) func(http.RoundTripper) http.RoundTripper {
	return func(rt http.RoundTripper) http.RoundTripper {
		return &CacheTransport{Wrapped: rt, cache: cache}
	}
}
