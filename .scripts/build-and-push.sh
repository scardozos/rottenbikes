#!/bin/bash

# Usage: ./.scripts/build-and-push.sh [remote_db_ip] [tag] [env_file]
# Example: ./.scripts/build-and-push.sh 172.233.115.22 latest .env.dev

REMOTE_DB_IP=$1
TAG=${2:-latest}
ENV_FILE=$3
DOCKER_REPO="monorailisland/rottenbikes"

if [ -z "$REMOTE_DB_IP" ]; then
    echo "Usage: $0 [remote_db_ip] [tag] [env_file]"
    echo "Example: $0 172.233.115.22 latest .env.dev"
    exit 1
fi

echo "--- Building Docker images ---"

# Build API (generic, no env hardcoding)
echo "Building API image..."
docker build --target api -t "${DOCKER_REPO}:api-${TAG}" .
if [ $? -ne 0 ]; then echo "API build failed"; exit 1; fi

# Build UI (generic, no env hardcoding)
echo "Building UI image..."
docker build --target ui -t "${DOCKER_REPO}:ui-${TAG}" .
if [ $? -ne 0 ]; then echo "UI build failed"; exit 1; fi

echo "--- Pushing images to Docker Hub ---"
docker push "${DOCKER_REPO}:api-${TAG}"
if [ $? -ne 0 ]; then echo "API push failed"; exit 1; fi

docker push "${DOCKER_REPO}:ui-${TAG}"
if [ $? -ne 0 ]; then echo "UI push failed"; exit 1; fi

echo "--- Running migrations against remote database ---"
# Load env file if provided to get DB credentials for migration
if [ -n "$ENV_FILE" ] && [ -f "$ENV_FILE" ]; then
    echo "Loading environment from $ENV_FILE for migrations..."
    export $(grep -v '^#' "$ENV_FILE" | xargs)
fi

if [ -z "$DB_USER" ] || [ -z "$DB_PASSWORD" ] || [ -z "$DB_NAME" ]; then
    echo "Error: DB_USER, DB_PASSWORD, and DB_NAME must be set in environment or provided via env_file"
    exit 1
fi

DB_DSN="postgres://$DB_USER:$DB_PASSWORD@$REMOTE_DB_IP:${DB_PORT:-5432}/$DB_NAME?sslmode=disable"
echo "Migrating database at $REMOTE_DB_IP..."
migrate -path internal/db/migrations -database "$DB_DSN" up

if [ $? -ne 0 ]; then 
    echo "Migration failed!"
    exit 1
fi

echo "--- Build and push complete! ---"
echo "Images pushed:"
echo "  - ${DOCKER_REPO}:api-${TAG}"
echo "  - ${DOCKER_REPO}:ui-${TAG}"
echo "Database migrations applied to $REMOTE_DB_IP"
