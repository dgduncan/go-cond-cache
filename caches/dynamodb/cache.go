package dynamodb

import (
	"context"

	db "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	gocondcache "github.com/dgduncan/go-cond-cache"
)

type Config struct {
	Endpoint               string
	Region                 string
	Table                  string
	DefaultCacheTTLSeconds int64
}

type Cache struct {
	db *db.Client

	table string
}

// GetHTTPResponse retrieves an http.Response from Redis for given key
func (c *Cache) Get(ctx context.Context, k string) (*gocondcache.CacheItem, error) {
	// 	key, err := attributevalue.Marshal(k)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	output, err := c.db.GetItem(ctx, &db.GetItemInput{
	// 		TableName: aws.String(c.table),
	// 		Key: map[string]types.AttributeValue{
	// 			"url": key,
	// 		},
	// 		ConsistentRead: aws.Bool(true),
	// 	})
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	if output.Item == nil {
	// 		return nil, errors.New("item not found")
	// 	}

	// 	cacheItem := new(gocondcache.CacheItem)
	// 	if err = attributevalue.UnmarshalMap(output.Item, cacheItem); err != nil {
	// 		return nil, err
	// 	}

	// 	if cacheItem.TTL < time.Now().Unix() {
	// 		return nil, errors.New("item has expired ttl")
	// 	}

	// 	r := bufio.NewReader(bytes.NewReader(cacheItem.Response))

	// 	res, err := http.ReadResponse(r, nil)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	return res, nil
	// }

	// // StoreHTTPResponse stores an http.Response in Redis
	// func (c *Cache) Set(ctx context.Context, k string, v *gocondcache.CacheItem) error {
	// 	if ttl <= 0 {
	// 		return nil
	// 	}

	// 	resBytes, err := httputil.DumpResponse(res, true)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	input := dynamodb.PutItemInput{
	// 		TableName: aws.String(c.table),
	// 		Item: map[string]*dynamodb.AttributeValue{
	// 			PrimaryKey: {
	// 				S: aws.String(key),
	// 			},
	// 			"response": {
	// 				B: resBytes,
	// 			},
	// 			"ttl": {
	// 				N: aws.String(strconv.Itoa(int(time.Now().Add(ttl).Unix()))),
	// 			},
	// 		},
	// 	}
	// 	_, err = c.client.PutItemWithContext(ctx, &input)
	return nil, nil
}

func New(ctx context.Context) (*Cache, error) {
	// client := db.NewFromConfig(aws.Config{})

	return &Cache{
		// db: client,
	}, nil

}
