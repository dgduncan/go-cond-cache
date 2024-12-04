package dynamodb

import (
	"bytes"
	"context"
	"encoding/gob"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	gocondcache "github.com/dgduncan/go-cond-cache"
	"github.com/dgduncan/go-cond-cache/caches"
)

type Config struct {
	DeleteExpiredItems bool // Controls if a the expired_at TTL property is put in the database to allow automatic deletion of expired items

	ItemExpiration time.Duration // How long a items stays valid in the database. This is independent of the expiration retrieved from the conditional response.
	Region         string
	Table          string
}

type Cache struct {
	client *dynamodb.Client

	table      string
	expiration time.Duration
	now        func() time.Time
}

type cacheItem struct {
	URL       string `json:"url"`
	Response  []byte `json:"response"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
	ExpiredAt int64  `json:"expired_at"`
}

// GetHTTPResponse retrieves an http.Response from Redis for given key
func (p *Cache) Get(ctx context.Context, k string) (*gocondcache.CacheItem, error) {
	key, err := attributevalue.Marshal(k)
	if err != nil {
		return nil, err
	}

	output, err := p.client.GetItem(ctx, &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"URL": key,
		},
		ConsistentRead: aws.Bool(true),
		TableName:      aws.String(p.table),
	})
	if err != nil {
		return nil, err
	}

	var item cacheItem
	if err := attributevalue.UnmarshalMap(output.Item, &item); err != nil {
		return nil, err
	}

	if p.now().UTC().Unix() >= item.ExpiredAt {
		return nil, nil
	}

	buff := bytes.NewBuffer(item.Response)
	dec := gob.NewDecoder(buff)

	var ci gocondcache.CacheItem
	if err := dec.Decode(&ci); err != nil {
		return nil, err
	}

	return &ci, nil
}

// StoreHTTPResponse stores an http.Response in Redis
func (c *Cache) Set(ctx context.Context, k string, v *gocondcache.CacheItem) error {
	createdAt := c.now().UTC()

	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	if err := enc.Encode(v); err != nil {
		return err
	}

	i := cacheItem{
		URL:       k,
		Response:  buff.Bytes(),
		CreatedAt: createdAt.Unix(),
		ExpiredAt: createdAt.Add(c.expiration).Unix(),
	}

	av, err := attributevalue.MarshalMap(i)
	if err != nil {
		return err
	}

	input := dynamodb.PutItemInput{
		TableName: aws.String(c.table),
		Item:      av,
	}

	_, err = c.client.PutItem(ctx, &input)
	return err
}

func NewDynamoDBCache(ctx context.Context, c *Config) (*Cache, error) {
	var itemExpiration time.Duration
	if c.ItemExpiration == 0 {
		itemExpiration = caches.DefaultExpiredDuration
	}
	awsConfig, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(c.Region))
	if err != nil {
		return nil, err
	}

	client := dynamodb.NewFromConfig(awsConfig)

	return &Cache{
		client: client,

		table:      c.Table,
		expiration: itemExpiration,
		now:        time.Now,
	}, nil
}
