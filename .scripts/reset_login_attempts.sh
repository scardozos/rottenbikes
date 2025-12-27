#!/bin/bash

# Usage: .scripts/reset_login_attempts.sh <email_or_username> <env_file>

IDENTIFIER="$1"
ENV_FILE="$2"

if [ -z "$IDENTIFIER" ] || [ -z "$ENV_FILE" ]; then
  echo "Usage: $0 <email_or_username> <env_file>"
  exit 1
fi

if [ ! -f "$ENV_FILE" ]; then
  echo "Error: Environment file '$ENV_FILE' not found."
  exit 1
fi

echo "Loading config from $ENV_FILE"
export $(grep -v '^#' "$ENV_FILE" | xargs)

# Construct DATABASE_URL if not set
if [ -z "$DATABASE_URL" ]; then
  DB_USER="${DB_USER:-rottenbikes}"
  DB_PASSWORD="${DB_PASSWORD:-rottenbikes}"
  DB_HOST="${DB_HOST:-localhost}"
  DB_PORT="${DB_PORT:-5432}"
  DB_NAME="${DB_NAME:-rottenbikes}"
  DB_SSLMODE="${DB_SSLMODE:-disable}"
  
  DATABASE_URL="postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=$DB_SSLMODE"
fi

echo "Resetting login attempts for user: $IDENTIFIER"

# Execute SQL
psql "$DATABASE_URL" -c "
DELETE FROM magic_links 
WHERE poster_id = (
    SELECT poster_id FROM posters WHERE email = '$IDENTIFIER' OR username = '$IDENTIFIER'
);
"

if [ $? -eq 0 ]; then
  echo "Successfully reset login attempts (if user existed)."
else
  echo "Failed to reset login attempts."
  exit 1
fi
