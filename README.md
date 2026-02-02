# user-prefs

A REST API for user preference CRUD, built with Go and backed by DynamoDB.

## Running locally

```bash
# Start DynamoDB Local and the app with Docker
docker compose up

# Or run directly (requires DYNAMODB_ENDPOINT and JWT_SECRET)
go run .
```

## Testing

```bash
go test ./...
```
