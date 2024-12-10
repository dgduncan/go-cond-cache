package dynamodb

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	gocondcache "github.com/dgduncan/go-cond-cache"
	"github.com/dgduncan/go-cond-cache/caches"
)

type Config struct {
	DeleteExpiredItems bool // Controls if a the expired_at TTL property is put in the database to allow automatic deletion of expired items

	ItemExpiration time.Duration // How long a items stays valid in the database. This is independent of the expiration retrieved from the conditional response.
	Table          string
}

type Cache struct {
	client *dynamodb.Client

	table      string
	expiration time.Duration
	now        func() time.Time
}

type cacheItem struct {
	URL       string `json:"url" dynamodbav:"url"`
	Response  []byte `json:"response" dynamodbav:"response"`
	CreatedAt int64  `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt int64  `json:"updated_at" dynamodbav:"updated_at"`
	ExpiredAt int64  `json:"expired_at" dynamodbav:"expired_at"`
}

// GetHTTPResponse retrieves an http.Response from Redis for given key
func (p *Cache) Get(ctx context.Context, k string) (*gocondcache.CacheItem, error) {
	key, err := attributevalue.Marshal(k)
	if err != nil {
		return nil, err
	}

	output, err := p.client.GetItem(ctx, &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"url": key,
		},
		ConsistentRead: aws.Bool(true),
		TableName:      aws.String(p.table),
	})
	if err != nil {
		return nil, err
	}

	if output.Item == nil {
		return nil, caches.ErrNoCacheItem
	}

	var item cacheItem
	if err := attributevalue.UnmarshalMap(output.Item, &item); err != nil {
		return nil, err
	}

	if p.now().UTC().Unix() >= item.ExpiredAt {
		return nil, caches.ErrCacheItemExpired
	}

	var ci gocondcache.CacheItem
	if err := gobDecode(item.Response, &ci); err != nil {
		return nil, err
	}

	return &ci, nil
}

// StoreHTTPResponse stores an http.Response in Redis
func (c *Cache) Set(ctx context.Context, k string, v *gocondcache.CacheItem) error {
	createdAt := c.now()

	encItem, err := gobEncode(v)
	if err != nil {
		return err
	}

	i := cacheItem{
		URL:       k,
		Response:  encItem,
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

func (c *Cache) Update(ctx context.Context, k string, expiration time.Time) error {
	key, err := attributevalue.Marshal(k)
	if err != nil {
		return err
	}

	expirationString := strconv.FormatInt(expiration.UTC().Unix(), 10) // converting to UTC may not be

	_, err = c.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(c.table),
		Key: map[string]types.AttributeValue{
			"url": key,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":expired_at": &types.AttributeValueMemberS{
				Value: *aws.String(expirationString),
			},
		},
		UpdateExpression: aws.String("SET expired_at = :expired_at"),
	})

	return err
}

func New(ctx context.Context, client *dynamodb.Client, config *Config) (*Cache, error) {
	if client == nil {
		return nil, caches.ValidationError{
			Reason: "nil client",
		}
	}

	var itemExpiration time.Duration
	if config.ItemExpiration == 0 {
		itemExpiration = caches.DefaultExpiredDuration
	} else {
		itemExpiration = config.ItemExpiration
	}

	return &Cache{
		client: client,

		table:      config.Table,
		expiration: itemExpiration,
		now:        time.Now,
	}, nil
}
