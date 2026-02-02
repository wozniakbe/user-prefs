package main

import "context"

// Store defines the persistence interface for user preferences.
type Store interface {
	GetAll(ctx context.Context, userID string) (map[string]string, error)
	Get(ctx context.Context, userID string, key string) (value string, found bool, err error)
	ReplaceAll(ctx context.Context, userID string, prefs map[string]string) error
	Update(ctx context.Context, userID string, prefs map[string]string) (merged map[string]string, err error)
	DeleteAll(ctx context.Context, userID string) error
	Delete(ctx context.Context, userID string, key string) error
}
