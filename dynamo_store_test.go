package main

import (
	"context"
	"os"
	"testing"
)

// Integration tests require DynamoDB Local running on DYNAMODB_ENDPOINT.
// Run with: DYNAMODB_ENDPOINT=http://localhost:8000 go test -run Integration ./...

func skipIfNoEndpoint(t *testing.T) {
	t.Helper()
	if os.Getenv("DYNAMODB_ENDPOINT") == "" {
		t.Skip("DYNAMODB_ENDPOINT not set; skipping integration test")
	}
}

func testStore(t *testing.T) *DynamoStore {
	t.Helper()
	cfg := Config{
		AWSRegion:       "us-east-1",
		DynamoEndpoint:  os.Getenv("DYNAMODB_ENDPOINT"),
		DynamoTableName: "user-preferences",
	}
	// Set dummy credentials for DynamoDB Local
	os.Setenv("AWS_ACCESS_KEY_ID", "test")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test")

	store, err := NewDynamoStore(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	return store
}

func TestIntegration_PutAndGetAll(t *testing.T) {
	skipIfNoEndpoint(t)
	store := testStore(t)
	ctx := context.Background()
	userID := "integration-test-user-1"

	// Clean up
	defer store.DeleteAll(ctx, userID)

	// Initially empty
	prefs, err := store.GetAll(ctx, userID)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if prefs != nil {
		t.Fatalf("expected nil prefs for new user, got %v", prefs)
	}

	// ReplaceAll
	err = store.ReplaceAll(ctx, userID, map[string]string{"theme": "dark", "lang": "en"})
	if err != nil {
		t.Fatalf("ReplaceAll: %v", err)
	}

	// GetAll
	prefs, err = store.GetAll(ctx, userID)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if prefs["theme"] != "dark" || prefs["lang"] != "en" {
		t.Fatalf("unexpected prefs: %v", prefs)
	}
}

func TestIntegration_GetSingle(t *testing.T) {
	skipIfNoEndpoint(t)
	store := testStore(t)
	ctx := context.Background()
	userID := "integration-test-user-2"

	defer store.DeleteAll(ctx, userID)

	store.ReplaceAll(ctx, userID, map[string]string{"theme": "light"})

	val, found, err := store.Get(ctx, userID, "theme")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found || val != "light" {
		t.Fatalf("expected theme=light found=true, got val=%s found=%v", val, found)
	}

	_, found, err = store.Get(ctx, userID, "missing")
	if err != nil {
		t.Fatalf("Get missing: %v", err)
	}
	if found {
		t.Fatal("expected found=false for missing key")
	}
}

func TestIntegration_Update(t *testing.T) {
	skipIfNoEndpoint(t)
	store := testStore(t)
	ctx := context.Background()
	userID := "integration-test-user-3"

	defer store.DeleteAll(ctx, userID)

	store.ReplaceAll(ctx, userID, map[string]string{"theme": "dark"})

	merged, err := store.Update(ctx, userID, map[string]string{"lang": "fr"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if merged["theme"] != "dark" || merged["lang"] != "fr" {
		t.Fatalf("unexpected merged: %v", merged)
	}
}

func TestIntegration_DeleteKey(t *testing.T) {
	skipIfNoEndpoint(t)
	store := testStore(t)
	ctx := context.Background()
	userID := "integration-test-user-4"

	defer store.DeleteAll(ctx, userID)

	store.ReplaceAll(ctx, userID, map[string]string{"theme": "dark", "lang": "en"})

	err := store.Delete(ctx, userID, "theme")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	prefs, _ := store.GetAll(ctx, userID)
	if _, exists := prefs["theme"]; exists {
		t.Fatal("expected theme to be deleted")
	}
	if prefs["lang"] != "en" {
		t.Fatal("expected lang to remain")
	}
}

func TestIntegration_DeleteAll(t *testing.T) {
	skipIfNoEndpoint(t)
	store := testStore(t)
	ctx := context.Background()
	userID := "integration-test-user-5"

	store.ReplaceAll(ctx, userID, map[string]string{"theme": "dark"})

	err := store.DeleteAll(ctx, userID)
	if err != nil {
		t.Fatalf("DeleteAll: %v", err)
	}

	prefs, _ := store.GetAll(ctx, userID)
	if prefs != nil {
		t.Fatalf("expected nil after DeleteAll, got %v", prefs)
	}
}
