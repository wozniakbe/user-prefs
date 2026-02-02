package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	ServerPort      string
	DynamoEndpoint  string
	DynamoTableName string
	JWTSecret       string
	JWTIssuer       string
	AWSRegion       string
	CORSAllowOrigin string
	LogLevel        slog.Level
	DevBypassAuth   bool
}

func LoadConfig() (Config, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	cfg := Config{
		ServerPort:      envOrDefault("SERVER_PORT", "8080"),
		DynamoEndpoint:  os.Getenv("DYNAMODB_ENDPOINT"),
		DynamoTableName: envOrDefault("DYNAMODB_TABLE_NAME", "user-preferences"),
		JWTSecret:       secret,
		JWTIssuer:       os.Getenv("JWT_ISSUER"),
		AWSRegion:       envOrDefault("AWS_REGION", "us-east-1"),
		CORSAllowOrigin: envOrDefault("CORS_ALLOW_ORIGIN", "*"),
		LogLevel:        parseLogLevel(os.Getenv("LOG_LEVEL")),
		DevBypassAuth:   strings.EqualFold(os.Getenv("DEV_BYPASS_AUTH"), "true"),
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
