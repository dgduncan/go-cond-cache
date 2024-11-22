package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"time"

	_ "github.com/lib/pq"

	gocondcache "github.com/dgduncan/go-cond-cache"
	"github.com/dgduncan/go-cond-cache/caches/postgres"
)

func main() {
	ctx := context.Background()

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	db, err := sql.Open("postgres", "postgresql://localhost:5455/postgresDB?user=postgresUser&password=postgresPW&sslmode=disable")
	if err != nil {
		panic(err)
	}

	c, err := postgres.NewPostgresCache(ctx, db, &postgres.Config{
		DeleteExpiredItems: true,
	})
	if err != nil {
		panic(err)
	}

	if err := c.Set(ctx, "hello/world", &gocondcache.CacheItem{
		ETAG:       "adsaa",
		Response:   []byte{},
		Expiration: time.Now().Add(10 * time.Minute),
	}); err != nil {
		println(err)
	}

	val, err := c.Get(ctx, "hello/world")
	if err != nil {
		panic(err)
	}

	fmt.Println(val)

	<-ctx.Done()

}
