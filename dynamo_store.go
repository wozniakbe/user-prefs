package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoStore implements Store using DynamoDB.
type DynamoStore struct {
	client    *dynamodb.Client
	tableName string
}

// NewDynamoStore creates a DynamoDB client and returns a DynamoStore.
func NewDynamoStore(ctx context.Context, cfg Config) (*DynamoStore, error) {
	var opts []func(*config.LoadOptions) error
	opts = append(opts, config.WithRegion(cfg.AWSRegion))

	if cfg.DynamoEndpoint != "" {
		opts = append(opts, config.WithBaseEndpoint(cfg.DynamoEndpoint))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	client := dynamodb.NewFromConfig(awsCfg)

	return &DynamoStore{
		client:    client,
		tableName: cfg.DynamoTableName,
	}, nil
}

func (s *DynamoStore) pk(userID string) string {
	return "USER#" + userID
}

func (s *DynamoStore) GetAll(ctx context.Context, userID string) (map[string]string, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: s.pk(userID)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("GetItem: %w", err)
	}

	if out.Item == nil {
		return nil, nil
	}

	return unmarshalPrefs(out.Item)
}

func (s *DynamoStore) Get(ctx context.Context, userID string, key string) (string, bool, error) {
	prefs, err := s.GetAll(ctx, userID)
	if err != nil {
		return "", false, err
	}

	val, found := prefs[key]
	return val, found, nil
}

func (s *DynamoStore) ReplaceAll(ctx context.Context, userID string, prefs map[string]string) error {
	now := time.Now().UTC().Format(time.RFC3339)

	prefsMap := make(map[string]types.AttributeValue, len(prefs))
	for k, v := range prefs {
		prefsMap[k] = &types.AttributeValueMemberS{Value: v}
	}

	item := map[string]types.AttributeValue{
		"PK":          &types.AttributeValueMemberS{Value: s.pk(userID)},
		"preferences": &types.AttributeValueMemberM{Value: prefsMap},
		"updatedAt":   &types.AttributeValueMemberS{Value: now},
		"createdAt":   &types.AttributeValueMemberS{Value: now},
	}

	_, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &s.tableName,
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("PutItem: %w", err)
	}

	return nil
}

func (s *DynamoStore) Update(ctx context.Context, userID string, prefs map[string]string) (map[string]string, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	// Build the update expression dynamically:
	// SET preferences.#k1 = :v1, preferences.#k2 = :v2, ..., updatedAt = :now
	exprNames := make(map[string]string, len(prefs))
	exprValues := make(map[string]types.AttributeValue, len(prefs)+1)

	updateExpr := "SET "
	i := 0
	for k, v := range prefs {
		nameKey := fmt.Sprintf("#k%d", i)
		valKey := fmt.Sprintf(":v%d", i)

		exprNames[nameKey] = k
		exprValues[valKey] = &types.AttributeValueMemberS{Value: v}

		if i > 0 {
			updateExpr += ", "
		}
		updateExpr += fmt.Sprintf("preferences.%s = %s", nameKey, valKey)
		i++
	}

	updateExpr += ", updatedAt = :now"
	exprValues[":now"] = &types.AttributeValueMemberS{Value: now}

	out, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: s.pk(userID)},
		},
		UpdateExpression:          &updateExpr,
		ExpressionAttributeNames:  exprNames,
		ExpressionAttributeValues: exprValues,
		ReturnValues:              types.ReturnValueAllNew,
	})
	if err != nil {
		return nil, fmt.Errorf("UpdateItem: %w", err)
	}

	return unmarshalPrefs(out.Attributes)
}

func (s *DynamoStore) DeleteAll(ctx context.Context, userID string) error {
	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: s.pk(userID)},
		},
	})
	if err != nil {
		return fmt.Errorf("DeleteItem: %w", err)
	}

	return nil
}

func (s *DynamoStore) Delete(ctx context.Context, userID string, key string) error {
	exprNames := map[string]string{"#key": key}
	updateExpr := "REMOVE preferences.#key"

	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: s.pk(userID)},
		},
		UpdateExpression:         &updateExpr,
		ExpressionAttributeNames: exprNames,
	})
	if err != nil {
		return fmt.Errorf("UpdateItem (REMOVE): %w", err)
	}

	return nil
}

// unmarshalPrefs extracts the preferences map from a DynamoDB item.
func unmarshalPrefs(item map[string]types.AttributeValue) (map[string]string, error) {
	prefsAttr, ok := item["preferences"]
	if !ok {
		return nil, nil
	}

	prefsMap, ok := prefsAttr.(*types.AttributeValueMemberM)
	if !ok {
		return nil, fmt.Errorf("preferences attribute is not a map")
	}

	result := make(map[string]string, len(prefsMap.Value))
	for k, v := range prefsMap.Value {
		sv, ok := v.(*types.AttributeValueMemberS)
		if !ok {
			continue
		}
		result[k] = sv.Value
	}

	return result, nil
}
