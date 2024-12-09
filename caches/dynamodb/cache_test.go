//go:build !integration

package dynamodb

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/dgduncan/go-cond-cache/caches"
)

func TestNewDynamoDBCache(t *testing.T) {
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
				Table:          "test-table",
				ItemExpiration: time.Hour,
			},
			expectedCache: nil,
			expectedErr:   caches.ErrValidation,
		},
		{
			name:   "zero item expiration uses default",
			client: &dynamodb.Client{},
			config: &Config{
				Table:          "test-table",
				ItemExpiration: 0,
			},
			expectedCache: &Cache{
				client:     &dynamodb.Client{},
				table:      "test-table",
				expiration: caches.DefaultExpiredDuration,
				now:        time.Now,
			},
			expectedErr: nil,
		},
		{
			name:   "custom item expiration",
			client: &dynamodb.Client{},
			config: &Config{
				Table:          "test-table",
				ItemExpiration: time.Hour,
			},
			expectedCache: &Cache{
				client:     &dynamodb.Client{},
				table:      "test-table",
				expiration: time.Hour,
				now:        time.Now,
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, err := New(context.Background(), tt.client, tt.config)

			if err != tt.expectedErr {
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
