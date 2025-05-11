package gocondcache

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/dgduncan/go-cond-cache/caches"
)

// type conditionalHeader string

const (
	headerCacheControl = "cache-control"
	headerETAG         = "etag"

	headerIfMatch     = "If-Match"
	headerIfNoneMatch = "If-None-Match"

	headerLastModified      = "Last-Modified"       //nolint
	headerIfMofifiedSince   = "If-Modified-Since"   //nolint
	headerIfUnmodifiedSince = "If-Unmodified-Since" //nolint
)

const (
	directiveCacheControlMaxAge = "max-age"
)

// CacheTransport implements http.RoundTripper and provides caching functionality
// for HTTP requests. It handles cache validation using ETags and manages cache
// expiration based on Cache-Control headers.
type CacheTransport struct {
	Wrapped http.RoundTripper

	cache  Cache
	logger *slog.Logger
	now    func() time.Time

	c Config
}

// RoundTrip implements http.RoundTripper interface and handles the caching logic
// for HTTP requests. It attempts to serve cached responses when valid, handles
// cache revalidation with ETags, and caches new responses when appropriate.
//
// The process follows these steps:
// 1. Checks for existing cache entry
// 2. Returns cached response if valid
// 3. Attempts revalidation if expired
// 4. Caches new responses with ETags
func (c *CacheTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	ctx := r.Context()

	// check if cached value exists within the cache
	item, err := c.cache.Get(ctx, caches.Key(*r))
	if err == nil { // cache hit
		c.logger.DebugContext(ctx, "cache item found", "url", r.URL.String())

		nr := bufio.NewReader(bytes.NewReader(item.Response))
		return http.ReadResponse(nr, nil)
	}

	// cache miss
	if errors.Is(err, caches.ErrCacheItemExpired) {
		// item has been found in the cache but is expired
		// check if item is still valid by adding etag to conditional request
		c.logger.DebugContext(ctx, "cache item expired, attempting revalidation",
			"url", r.URL.String(),
			"expiration", item.Expiration.Format(time.RFC3339))
		r.Header.Add(headerIfNoneMatch, item.ETAG)
	} else {
		c.logger.DebugContext(ctx, "cache item not found", "url", r.URL.String())
	}

	resp, transportError := c.Wrapped.RoundTrip(r)
	if transportError != nil {
		return resp, transportError
	}

	if resp.StatusCode != http.StatusPreconditionFailed && (resp.StatusCode < 200 || resp.StatusCode > 399) {
		return resp, transportError
	}

	// re-validation sucesfull
	if resp.StatusCode == http.StatusNotModified {
		// cache item as been revalidated as the response is 304
		c.logger.DebugContext(ctx, "cache item successfully revalidated", "url", r.URL.String())
		maxAge := getTimeToCache(resp, c.c.DomainOverrides)

		c.logger.DebugContext(ctx, "updating cache item", "url", r.URL.String(), "expiration", c.now().UTC().Add(maxAge).Format(time.RFC3339))
		if err := c.cache.Update(ctx, caches.Key(*resp.Request), c.now().UTC().Add(maxAge)); err != nil {
			return resp, errors.Join(err, transportError) // return original http response and error
		}

		nr := bufio.NewReader(bytes.NewReader(item.Response))
		return http.ReadResponse(nr, nil)
	}

	// check if response contains conditional request header i.e etag
	if getETAGHeader(resp) == "" { // if no etag header is found, we don't cache the response
		c.logger.DebugContext(ctx, "no etag header found, not caching response", "url", r.URL.String())
		return resp, transportError
	}

	// cache the response
	maxAge := getTimeToCache(resp, c.c.DomainOverrides)
	c.logger.DebugContext(ctx, "caching response", "url", r.URL.String(), "expiration", c.now().UTC().Add(maxAge))
	resBytes, _ := httputil.DumpResponse(resp, true)
	if err := c.cache.Set(ctx, caches.Key(*resp.Request), &CacheItem{
		ETAG:       resp.Header.Get(headerETAG),
		Response:   resBytes,
		Expiration: c.now().UTC().Add(maxAge),
	}); err != nil {
		c.logger.WarnContext(ctx, "error caching response", "error", err)
	}

	return resp, transportError
}

func getTimeToCache(r *http.Response, c []DomainOverride) time.Duration {
	// check to see if any domain overrides exist
	for _, v := range c {
		if strings.HasPrefix(r.Request.URL.Host+r.Request.URL.Path, v.URI) {
			slog.DebugContext(context.Background(), "caching override found")
			return v.Duration
		}
	}

	// Get the Cache-Control header value
	return getMaxAge(r)
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
		if strings.HasPrefix(directive, string(directiveCacheControlMaxAge)) {
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

// New creates a transport middleware that adds caching capabilities to an HTTP RoundTripper.
// It implements conditional request caching using ETags and enables cache revalidation.
//
// The middleware uses the provided Cache implementation for storing and retrieving cached responses.
// If the 'now' function is nil, time.Now will be used as the default time provider.
// If the 'logger' is nil, a no-op logger writing to io.Discard will be used.
//
// The returned function wraps the given http.RoundTripper with caching functionality:
//   - Caches responses that contain ETag headers
//   - Handles cache revalidation using If-None-Match headers
//   - Respects Cache-Control max-age directives for expiration
//   - Logs cache operations when a logger is provided
func New(cache Cache, opts *Config, now func() time.Time, logger *slog.Logger) func(http.RoundTripper) http.RoundTripper {
	nowFunc := now
	if nowFunc == nil {
		nowFunc = time.Now
	}

	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	c := Config{}
	if opts == nil {
		c = DefaultConfig()
	} else {
		c = *opts
	}

	return func(rt http.RoundTripper) http.RoundTripper {
		return &CacheTransport{Wrapped: rt, cache: cache, now: nowFunc, logger: logger, c: c}
	}
}
