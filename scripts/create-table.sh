#!/usr/bin/env bash
set -euo pipefail

ENDPOINT="${DYNAMODB_ENDPOINT:-http://localhost:8000}"
TABLE_NAME="${DYNAMODB_TABLE_NAME:-user-preferences}"
REGION="${AWS_REGION:-us-east-1}"

echo "Creating table '${TABLE_NAME}' at ${ENDPOINT}..."

aws dynamodb create-table \
  --endpoint-url "${ENDPOINT}" \
  --region "${REGION}" \
  --table-name "${TABLE_NAME}" \
  --attribute-definitions AttributeName=PK,AttributeType=S \
  --key-schema AttributeName=PK,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  2>/dev/null && echo "Table created." || echo "Table already exists or creation failed."
