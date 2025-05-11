package gocondcache

import "time"

type Config struct {
	// DefaultExpiration is used when the specified cache directive is not present
	// in the response headers. Zero means no default caching.
	DefaultExpiration time.Duration

	// DomainOverrides allow for users to override the caching-directive responses from
	// upstream servers and cache for an arbitrary amount of time. Once expired, will attempt
	// to revalidate the cached item with a conditional request. If upstream server does not return
	// a cache-control header Expires, or Etag header, caching will be completely bypassed.
	DomainOverrides []DomainOverride
}

type DomainOverride struct {
	URI string // eg. www.misbehaving_caching_domain.com/

	Duration time.Duration // eg. 1H
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() Config {
	return Config{
		DefaultExpiration: 0, // No caching by default if StrictDirectiveHandling is set to false
		DomainOverrides:   nil,
	}
}
