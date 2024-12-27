package postgres

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"encoding/gob"
	"errors"
	"log"
	"time"

	_ "github.com/lib/pq"

	gocondcache "github.com/dgduncan/go-cond-cache"
	"github.com/dgduncan/go-cond-cache/caches"
)

var (
	// ErrPingFailed is returned if the initial ping to the database returns an error
	ErrPingFailed = errors.New("ping returned error")
)

var (
	//go:embed create_table.sql
	queryCreateTable string
	//go:embed delete_expired.sql
	queryDeleteExpired string
	//go:embed fetch_by_id.sql
	queryFetchByID string
	//go:embed insert_item.sql
	queryInsertItem string
	//go:embed update_item.sql
	queryUpdateItem string
)

// Config defines the configuration options for the PostgreSQL cache implementation.
type Config struct {
	// DeleteExpiredItems enables automatic cleanup of expired cache entries
	// through a background task.
	DeleteExpiredItems bool

	// ExpiredTaskTimer defines the interval at which the cleanup task runs.
	// Shorter durations may impact database performance.
	ExpiredTaskTimer time.Duration

	// ItemExpiration defines how long items remain valid in the database.
	// This is separate from the expiration time derived from conditional response headers.
	ItemExpiration time.Duration
}

// Cache implements the gocondcache.Cache interface using PostgreSQL as the storage backend.
// It provides thread-safe operations for storing and retrieving cached HTTP responses.
type Cache struct {
	db *sql.DB

	now func() time.Time
}

// Get retrieves a cache item from PostgreSQL by its key. It returns the cached item
// if found and not expired, or an appropriate error otherwise.
// Returns caches.ErrNoCacheItem if the item doesn't exist.
func (p *Cache) Get(ctx context.Context, k string) (*gocondcache.CacheItem, error) {
	stmt, err := p.db.PrepareContext(ctx, queryFetchByID)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRowContext(ctx, k, p.now().UTC())
	if err := row.Err(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, caches.ErrNoCacheItem
		}
		return nil, err
	}

	var url string
	var response []byte
	if err := row.Scan(&url, &response); err != nil {
		return nil, err
	}

	buff := bytes.NewBuffer(response)
	dec := gob.NewDecoder(buff)

	var item gocondcache.CacheItem
	if err := dec.Decode(&item); err != nil {
		return nil, err
	}

	return &item, nil
}

// Set stores a new cache item in PostgreSQL with the provided key and value.
// It handles the serialization of the cache item using gob encoding.
func (p *Cache) Set(ctx context.Context, k string, v *gocondcache.CacheItem) error {
	stmt, err := p.db.PrepareContext(ctx, queryInsertItem)
	if err != nil {
		return err
	}
	defer stmt.Close()

	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	if err := enc.Encode(v); err != nil {
		return err
	}

	_, err = stmt.ExecContext(ctx, k, buff.Bytes(), p.now().UTC().Add(caches.DefaultExpiredDuration))
	return err
}

// Update modifies the expiration time of an existing cache item in PostgreSQL.
// This is typically used when a cached response is revalidated with the origin server
func (bc *Cache) Update(ctx context.Context, key string, expiration time.Time) error {
	stmt, err := bc.db.PrepareContext(ctx, queryUpdateItem)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, key, expiration, bc.now().UTC())
	return err
}

func createTable(ctx context.Context, db *sql.DB) error {
	stmt, err := db.PrepareContext(ctx, queryCreateTable)
	if err != nil {
		return err
	}

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		return err
	}

	return nil
}

func deleteExpiredItems(ctx context.Context, db *sql.DB) error {
	stmt, err := db.PrepareContext(ctx, queryDeleteExpired)
	if err != nil {
		return err
	}

	_, err = stmt.ExecContext(ctx)
	return err
}

func expiredTask(ctx context.Context, db *sql.DB) {
	t := time.NewTimer(5 * time.Second)

	for {
		select {
		case <-ctx.Done():
			log.Println("context is done")
			return
		case <-t.C:
			if err := deleteExpiredItems(ctx, db); err != nil {
				log.Println(err)
			}
			_ = t.Reset(5 * time.Second)
		}
	}
}

// New creates a new PostgreSQL cache instance with the provided configuration.
// It verifies the database connection, creates the necessary table structure, and
// optionally starts the cleanup task for expired items.
//
// Returns an error if:
// - The database connection test fails
// - Table creation fails
// - Configuration validation fails
func New(ctx context.Context, db *sql.DB, config *Config) (*Cache, error) {
	if err := db.PingContext(ctx); err != nil {
		return nil, errors.Join(ErrPingFailed, err)
	}

	if err := createTable(ctx, db); err != nil {
		return nil, err
	}

	if config != nil {
		if config.DeleteExpiredItems {
			go expiredTask(ctx, db)
		}
	}

	return &Cache{
		db: db,

		now: time.Now,
	}, nil
}
