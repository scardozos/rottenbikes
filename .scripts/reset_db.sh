#!/bin/bash
set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <env_file>"
    echo "Example: $0 .env.dev"
    exit 1
fi

ENV_FILE="$1"
# Extract env extension if possible (e.g. .env.dev -> dev), otherwise use full name
ENV_SUFFIX="${ENV_FILE##*.}"
if [ "$ENV_SUFFIX" = "$ENV_FILE" ]; then
    ENV_SUFFIX="custom"
fi
BACKUP_FILE="backup_data_${ENV_SUFFIX}.sql"

echo "--- Loading environment from $ENV_FILE ---"
if [ -f "$ENV_FILE" ]; then
    set -o allexport
    source "$ENV_FILE"
    set +o allexport
else
    echo "Error: $ENV_FILE not found."
    exit 1
fi

# Sanitize/Encode password for DSN
# We use Python for encoding to handle special chars safe for the connection string
DB_PASSWORD="${DB_PASSWORD:-rottenbikes}"
DB_PASSWORD_ENCODED=$(python3 -c "import urllib.parse, sys; print(urllib.parse.quote(sys.argv[1]))" "$DB_PASSWORD")

DB_USER="${DB_USER:-rottenbikes}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-rottenbikes}"

# Construct DSN
DB_DSN="postgres://$DB_USER:$DB_PASSWORD_ENCODED@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=disable"

# For pg_dump/psql which use PGPASSWORD
export PGPASSWORD="$DB_PASSWORD"

echo "--- 1. Backing up data from $DB_HOST ---"
# --data-only to avoid schema conflicts (since we are resetting schema)
# --column-inserts to be safer if column order changed
# --disable-triggers to avoid foreign key issues during validation/restore
pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
    --data-only --column-inserts --disable-triggers \
    -f "$BACKUP_FILE"

echo "Backup saved to $BACKUP_FILE"

echo "--- 2. Resetting Database (Dropping public schema) ---"
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

echo "--- 3. Re-applying Migrations ---"
echo "Applying migrations from internal/db/migrations..."
migrate -path internal/db/migrations -database "$DB_DSN" up

echo "--- 4. Restoring Data ---"
# We ignore errors solely because some data might violate new constraints or duplicate what migration inserted (if any)
# But standard restoring should stop on error usually.
# Use ON_ERROR_STOP=1 to fail fast or remove it to try best effort.
# Given it's a reset, let's try to restore and see.
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$BACKUP_FILE"

echo "--- Done! ---"
echo "You can check the database now."
