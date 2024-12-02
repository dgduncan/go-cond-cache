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
)

type Config struct {
	DeleteExpiredItems bool          // Controls if background task runs to delete expired items from the database
	ExpiredTaskTimer   time.Duration // How often the background rask runs to delete expired items. Shorter durations can cause unecessary DB overhead

	ItemExpiration time.Duration // How long a items stays valid in the database. This is independent of the expiration retrieved from the conditional response.
}

type Cache struct {
	db *sql.DB

	now func() time.Time
}

func (p *Cache) Get(ctx context.Context, k string) (*gocondcache.CacheItem, error) {
	stmt, err := p.db.PrepareContext(ctx, queryFetchByID)
	if err != nil {
		return nil, err
	}

	row := stmt.QueryRowContext(ctx, k, p.now().UTC())
	if err := row.Err(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, gocondcache.ErrNotFound
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

func (p *Cache) Set(ctx context.Context, k string, v *gocondcache.CacheItem) error {
	stmt, err := p.db.PrepareContext(ctx, queryInsertItem)
	if err != nil {
		return err
	}

	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	if err := enc.Encode(v); err != nil {
		return err
	}

	_, err = stmt.ExecContext(ctx, k, buff.Bytes(), p.now().UTC().Add(caches.DefaultExpiredDuration))
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

func NewPostgresCache(ctx context.Context, db *sql.DB, config *Config) (*Cache, error) {
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
