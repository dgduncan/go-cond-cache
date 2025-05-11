package gocondcache

import "time"

type Config struct {
	// DomainOverrides allow for users to override the caching-directive responses from
	// upstream servers and cache for an arbitrary amount of time. Once expired, will attempt
	// to revalidate the cached item with a conditional request. If upstream server does not return
	// a cache-control header Expires, or Etag header, caching will be completely bypassed.
	DomainOverrides []DomainOverride
}

type DomainOverride struct {
	URI string // eg. misbehaving_caching_domain.com

	Duration time.Duration // eg. 1H
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() Config {
	return Config{
		DomainOverrides: nil,
	}
}
