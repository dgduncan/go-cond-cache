package gocondcache_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gocondcache "github.com/dgduncan/go-cond-cache"
	"github.com/dgduncan/go-cond-cache/caches/local"
)

func testTime() time.Time {
	return time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
}

func TestLastModifiedCaching(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		serverResponse       func(w http.ResponseWriter, r *http.Request)
		expectedCacheHit     bool
		expectedLastModified string
	}{
		{
			name: "response with Last-Modified header gets cached",
			serverResponse: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
				w.Header().Set("Cache-Control", "max-age=86400") // 24 hours
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("cached content"))
			},
			expectedCacheHit:     true,
			expectedLastModified: "Wed, 21 Oct 2015 07:28:00 GMT",
		},
		{
			name: "response with ETag and Last-Modified gets cached",
			serverResponse: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
				w.Header().Set("ETag", `"abc123"`)
				w.Header().Set("Cache-Control", "max-age=86400") // 24 hours
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("cached content"))
			},
			expectedCacheHit:     true,
			expectedLastModified: "Wed, 21 Oct 2015 07:28:00 GMT",
		},
		{
			name: "response without conditional headers not cached",
			serverResponse: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Cache-Control", "max-age=86400") // 24 hours
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("not cached content"))
			},
			expectedCacheHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create test server
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			// Create cache and transport with fixed time
			baseTime := testTime()
			cache := local.NewBasicCacheWithTimeFunc(func() time.Time { return baseTime })
			transport := gocondcache.New(
				&cache,
				nil,
				func() time.Time { return baseTime },
				slog.New(slog.NewTextHandler(io.Discard, nil)),
			)(http.DefaultTransport)

			client := &http.Client{Transport: transport}

			// Make first request
			resp1, err := client.Get(server.URL)
			if err != nil {
				t.Fatalf("first request failed: %v", err)
			}
			resp1.Body.Close()

			// Check if item was cached (use the same time for cache checks)
			ctx := context.Background()
			cacheKey := fmt.Sprintf("GET#%s", server.URL)

			item, err := cache.Get(ctx, cacheKey)
			if tt.expectedCacheHit {
				if err != nil {
					t.Fatalf("expected cache hit but got error: %v", err)
				}
				if tt.expectedLastModified != "" {
					if item.LastModified == nil {
						t.Fatal("expected LastModified to be set")
					}
					if item.LastModified.Format(http.TimeFormat) != tt.expectedLastModified {
						t.Errorf("expected LastModified %s, got %s",
							tt.expectedLastModified, item.LastModified.Format(http.TimeFormat))
					}
				}
			} else {
				if err == nil {
					t.Fatal("expected cache miss but got cache hit")
				}
			}
		})
	}
}

func TestIfModifiedSinceConditionalRequest(t *testing.T) {
	t.Parallel()

	lastModifiedTime := time.Date(2015, 10, 21, 7, 28, 0, 0, time.UTC)
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Check for If-Modified-Since header
		ifModifiedSince := r.Header.Get("If-Modified-Since")
		if ifModifiedSince != "" {
			// Parse the If-Modified-Since header
			ifModTime, err := time.Parse(http.TimeFormat, ifModifiedSince)
			if err == nil && !lastModifiedTime.After(ifModTime) {
				// Resource hasn't been modified
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		// Return the resource with Last-Modified header
		w.Header().Set("Last-Modified", lastModifiedTime.Format(http.TimeFormat))
		w.Header().Set("Cache-Control", "max-age=1") // Short cache time
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("content"))
	}))
	defer server.Close()

	// Create cache and transport with mutable time
	currentTime := testTime()
	timeFunc := func() time.Time { return currentTime }

	cache := local.NewBasicCacheWithTimeFunc(timeFunc)
	transport := gocondcache.New(
		&cache,
		nil,
		timeFunc,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)(http.DefaultTransport)

	client := &http.Client{Transport: transport}

	// First request - should cache the response
	resp1, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	defer resp1.Body.Close()

	if requestCount != 1 {
		t.Errorf("expected 1 request to server, got %d", requestCount)
	}

	// Move time forward to expire the cache
	currentTime = currentTime.Add(2 * time.Second)

	// Second request - should send If-Modified-Since and get 304
	resp2, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	defer resp2.Body.Close()

	if requestCount != 2 {
		t.Errorf("expected 2 requests to server, got %d", requestCount)
	}

	// Verify the cached response is served
	body, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if string(body) != "content" {
		t.Errorf("expected cached content, got %s", string(body))
	}
}

func TestLastModifiedWithETagPriority(t *testing.T) {
	t.Parallel()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Check for both conditional headers
		ifNoneMatch := r.Header.Get("If-None-Match")
		ifModifiedSince := r.Header.Get("If-Modified-Since")

		// When both are present, both should be sent
		if requestCount > 1 {
			if ifNoneMatch == "" || ifModifiedSince == "" {
				t.Errorf("expected both If-None-Match and If-Modified-Since headers on revalidation")
			}
		}

		if ifNoneMatch == `"etag123"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		w.Header().Set("ETag", `"etag123"`)
		w.Header().Set("Cache-Control", "max-age=1")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("content with both headers"))
	}))
	defer server.Close()

	// Create cache and transport with mutable time
	currentTime := testTime()
	timeFunc := func() time.Time { return currentTime }

	cache := local.NewBasicCacheWithTimeFunc(timeFunc)
	transport := gocondcache.New(
		&cache,
		nil,
		timeFunc,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)(http.DefaultTransport)

	client := &http.Client{Transport: transport}

	// First request
	resp1, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	resp1.Body.Close()

	// Expire cache
	currentTime = currentTime.Add(2 * time.Second)

	// Second request - should send both headers and get 304
	resp2, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	resp2.Body.Close()

	if requestCount != 2 {
		t.Errorf("expected 2 requests to server, got %d", requestCount)
	}
}

func TestLastModifiedParsingErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		lastModified   string
		shouldBeCached bool
	}{
		{
			name:           "valid Last-Modified header",
			lastModified:   "Wed, 21 Oct 2015 07:28:00 GMT",
			shouldBeCached: true,
		},
		{
			name:           "invalid Last-Modified header format",
			lastModified:   "invalid-date-format",
			shouldBeCached: false,
		},
		{
			name:           "empty Last-Modified header",
			lastModified:   "",
			shouldBeCached: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tt.lastModified != "" {
					w.Header().Set("Last-Modified", tt.lastModified)
				}
				w.Header().Set("Cache-Control", "max-age=86400") // 24 hours
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("test content"))
			}))
			defer server.Close()

			baseTime := testTime()
			cache := local.NewBasicCacheWithTimeFunc(func() time.Time { return baseTime })
			transport := gocondcache.New(
				&cache,
				nil,
				func() time.Time { return baseTime },
				slog.New(slog.NewTextHandler(io.Discard, nil)),
			)(http.DefaultTransport)

			client := &http.Client{Transport: transport}

			resp, err := client.Get(server.URL)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			resp.Body.Close()

			// Check cache
			ctx := context.Background()
			cacheKey := fmt.Sprintf("GET#%s", server.URL)

			_, err = cache.Get(ctx, cacheKey)
			if tt.shouldBeCached && err != nil {
				t.Errorf("expected response to be cached, but got error: %v", err)
			}
			if !tt.shouldBeCached && err == nil {
				t.Error("expected response not to be cached, but it was cached")
			}
		})
	}
}
