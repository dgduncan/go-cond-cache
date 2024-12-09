//go:build integration

package dynamodb

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	gocondcache "github.com/dgduncan/go-cond-cache"
	"github.com/stretchr/testify/assert"
)

func setup(t *testing.T) (*dynamodb.Client, error) {
	t.Log("setup called")

	awsconfig, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("local"))
	if err != nil {
		return nil, err
	}

	c := dynamodb.NewFromConfig(awsconfig)

	if err := createTable(context.Background(), c); err != nil {
		return nil, err
	}

	if err := putCacheItem(t, c); err != nil {
		return nil, err
	}

	key, err := attributevalue.Marshal("hello")
	if err != nil {
		return nil, err
	}
	output, err := c.GetItem(context.Background(), &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"url": key,
		},
		ConsistentRead: aws.Bool(true),
		TableName:      aws.String("test"),
	})
	if err != nil {
		return nil, err
	}

	fmt.Println(output.Item == nil)

	return c, nil
}

func cleanup(t *testing.T, c *dynamodb.Client) {
	t.Log("cleanup called")

	output, err := c.ListTables(context.Background(), &dynamodb.ListTablesInput{})
	if err != nil {
		t.Log(err)
		return
	}

	for _, v := range output.TableNames {
		if _, err := c.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{
			TableName: aws.String(v),
		}); err != nil {
			t.Log(err)
		}
	}
}

func putCacheItem(t *testing.T, c *dynamodb.Client) error {
	ci := gocondcache.CacheItem{
		ETAG:       "etag",
		Response:   []byte{},
		Expiration: time.Now().Add(1 * time.Second),
	}

	b, _ := gobEncode(ci)

	i := cacheItem{
		URL:       "hello",
		Response:  b,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		ExpiredAt: time.Now().Add(1 * time.Minute).Unix(),
	}

	av, err := attributevalue.MarshalMap(i)
	if err != nil {
		return err
	}

	input := dynamodb.PutItemInput{
		TableName: aws.String("test"),
		Item:      av,
	}

	_, err = c.PutItem(context.Background(), &input)
	return err
}

func TestGetIntegration(t *testing.T) {
	c, err := setup(t)
	if err != nil {
		t.Log(err)
		t.FailNow()
		return
	}

	t.Cleanup(func() {
		cleanup(t, c)
	})

	tests := []struct {
		name        string
		client      *dynamodb.Client
		key         string
		cacheHit    bool
		expectedErr error
	}{
		{
			name:     "golden path - cache hit",
			client:   c,
			cacheHit: true,
			key:      "hello",
		},
		{
			name:     "golden path - cache miss",
			client:   c,
			cacheHit: false,
			key:      "key-miss",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			d, err := New(ctx, c, &Config{
				Table: "test",

				ItemExpiration: 1 * time.Minute,
			})
			if err != nil {
				t.Fail()
				return
			}

			resp, err := d.Get(ctx, tt.key)
			if tt.expectedErr != nil {
				assert.Error(t, err)
			}

			if tt.cacheHit {
				assert.NotNil(t, resp)
			} else {
				assert.Nil(t, resp)
			}

		})
	}
}
