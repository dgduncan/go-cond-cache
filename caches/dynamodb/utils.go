package dynamodb

import (
	"bytes"
	"context"
	"encoding/gob"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func createTable(ctx context.Context, client *dynamodb.Client) error { //nolint:unused // not used at the moment
	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String("test"),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("url"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("url"),
				KeyType:       types.KeyTypeHash,
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(DefaultReadCapacityUnits),
			WriteCapacityUnits: aws.Int64(DefaultWriteCapacityUnits),
		},
	})

	return err
}

func gobEncode(v any) ([]byte, error) {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

func gobDecode(i []byte, t any) error {
	buff := bytes.NewBuffer(i)
	dec := gob.NewDecoder(buff)

	return dec.Decode(t)
}
