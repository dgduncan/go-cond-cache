//go:build !integration

package dynamodb

import (
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/dgduncan/go-cond-cache/caches"
)

const (
	tableName = "test-table"
)

var (
	testingTime = func() time.Time {
		return time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	}
)

func TestNewDynamoDBCache(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		client        *dynamodb.Client
		config        *Config
		expectedCache *Cache
		expectedErr   error
	}{
		{
			name:   "nil client returns error",
			client: nil,
			config: &Config{
				Table:          tableName,
				ItemExpiration: time.Hour,
			},
			expectedCache: nil,
			expectedErr: caches.ValidationError{
				Reason: "nil client",
			},
		},
		{
			name:   "zero item expiration uses default",
			client: &dynamodb.Client{},
			config: &Config{
				Table:          tableName,
				ItemExpiration: 0,
			},
			expectedCache: &Cache{
				client:     &dynamodb.Client{},
				table:      tableName,
				expiration: caches.DefaultExpiredDuration,
				now:        testingTime,
			},
			expectedErr: nil,
		},
		{
			name:   "custom item expiration",
			client: &dynamodb.Client{},
			config: &Config{
				Table:          tableName,
				ItemExpiration: time.Hour,
			},
			expectedCache: &Cache{
				client:     &dynamodb.Client{},
				table:      tableName,
				expiration: time.Hour,
				now:        testingTime,
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cache, err := New(tt.client, tt.config)

			if errors.Is(err, tt.expectedErr) {
				t.Errorf("expected error %v, got %v", tt.expectedErr, err)
			}

			if tt.expectedCache == nil {
				if cache != nil {
					t.Error("expected nil cache")
				}
				return
			}

			if cache.table != tt.expectedCache.table {
				t.Errorf("expected table %s, got %s", tt.expectedCache.table, cache.table)
			}

			if cache.expiration != tt.expectedCache.expiration {
				t.Errorf("expected expiration %v, got %v", tt.expectedCache.expiration, cache.expiration)
			}
		})
	}
}
