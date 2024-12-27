# go-cond-cache

A flexible caching solution for conditional HTTP requests (If-None-Match, If-Modified-Since) with support for multiple storage backends.

## Overview

This library provides a caching mechanism for handling conditional HTTP requests efficiently. It supports three different storage backends:
- Local Memory Cache
- PostgreSQL
- Amazon DynamoDB

The cache helps reduce unnecessary data transfer and server load by properly handling ETags and Last-Modified headers.

## Features

- Multiple storage backend options
- Thread-safe operations
- Support for ETags and Last-Modified headers
- Configurable TTL (Time To Live)
- Easy integration with existing applications
- Concurrent access handling
- Automatic cache invalidation

## Installation

```bash
go get github.com/dgduncan/go-cond-cache
```

## Usage

### Basic Example

```go
import (
    "github.com/dgduncan/go-cond-cache"
)

// Initialize a local cache
cache, err := conditionalcache.NewLocalCache(conditionalcache.Config{
    TTL: time.Hour * 24,
})

// Initialize a PostgreSQL cache
pgCache, err := conditionalcache.NewPostgresCache(conditionalcache.PostgresConfig{
    ConnString: "postgres://user:pass@localhost:5432/dbname",
    TableName:  "conditional_cache",
})

// Initialize a DynamoDB cache
dynamoCache, err := conditionalcache.NewDynamoCache(conditionalcache.DynamoConfig{
    Region:    "us-west-2",
    TableName: "conditional-cache",
})
```

### Storing and Retrieving Cache Entries

```go
// Store a cache entry
err := cache.Set(ctx, "key", CacheEntry{
    ETag:         "abc123",
    LastModified: time.Now(),
    Data:         []byte("cached data"),
})

// Get a cache entry
entry, err := cache.Get(ctx, "key")
```

## Storage Backend Configuration

### Local Cache

```go
config := conditionalcache.Config{
    TTL:        time.Hour * 24,
    MaxEntries: 1000,
}
```

### PostgreSQL Cache

```go
config := conditionalcache.PostgresConfig{
    ConnString:      "postgres://user:pass@localhost:5432/dbname",
    TableName:       "conditional_cache",
    TTL:            time.Hour * 24,
    CleanupInterval: time.Hour,
}
```

### DynamoDB Cache

```go
config := conditionalcache.DynamoConfig{
    Region:          "us-west-2",
    TableName:       "conditional-cache",
    TTL:            time.Hour * 24,
    ReadCapacity:   5,
    WriteCapacity:  5,
}
```

## Schema Definitions

### PostgreSQL Table Schema

```sql
CREATE TABLE conditional_cache (
    key TEXT PRIMARY KEY,
    etag TEXT,
    last_modified TIMESTAMP,
    data BYTEA,
    created_at TIMESTAMP,
    expires_at TIMESTAMP
);
```

### DynamoDB Table Schema

```json
{
    "TableName": "conditional-cache",
    "KeySchema": [
        {
            "AttributeName": "key",
            "KeyType": "HASH"
        }
    ],
    "AttributeDefinitions": [
        {
            "AttributeName": "key",
            "AttributeType": "S"
        }
    ]
}
```

## Error Handling

The library returns detailed errors that can be handled using the provided error types:

```go
switch err.(type) {
case *conditionalcache.NotFoundError:
    // Handle cache miss
case *conditionalcache.ExpiredError:
    // Handle expired entry
case *conditionalcache.StorageError:
    // Handle storage-related errors
}
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by RFC 7232 (Conditional Requests)
- Built with best practices for caching in distributed systems
```

This README provides a comprehensive overview of your project, including:
- Clear installation instructions
- Usage examples for all three cache types
- Configuration options
- Schema definitions
- Error handling examples
- Contributing guidelines
- License information

You can customize it further based on your specific implementation details, additional features, or requirements.
