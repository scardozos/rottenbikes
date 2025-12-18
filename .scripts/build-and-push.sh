#!/bin/bash

# Usage: ./.scripts/build-and-push.sh [remote_ip] [tag] [env_file]
# Example: ./.scripts/build-and-push.sh 192.168.1.100 v1.0.0 .env.dev

REMOTE_IP=$1
TAG=${2:-latest}
ENV_FILE=${3:-.env.dev}
REMOTE_USER="root"

if [ -z "$REMOTE_IP" ]; then
    echo "Usage: $0 [remote_ip] [tag] [env_file]"
    exit 1
fi

if [ ! -f "$ENV_FILE" ]; then
    echo "Environment file $ENV_FILE not found!"
    exit 1
fi

echo "--- Loading environment from $ENV_FILE ---"
# Source the env file, stripping comments and exported keywords if present
# But for simplicity, we assume simple key=value format
export $(grep -v '^#' "$ENV_FILE" | xargs)

echo "--- Building API and UI images ---"

# Build API
docker build --target api -t "rottenbikes-api:$TAG" .
if [ $? -ne 0 ]; then echo "API build failed"; exit 1; fi

# Build UI with API URL from env
echo "Building UI with API_URL: $EXPO_PUBLIC_API_URL"
docker build --target ui --build-arg EXPO_PUBLIC_API_URL="$EXPO_PUBLIC_API_URL" -t "rottenbikes-ui:$TAG" .
if [ $? -ne 0 ]; then echo "UI build failed"; exit 1; fi

echo "--- Saving images to tarballs ---"
docker save "rottenbikes-api:$TAG" > api.tar
docker save "rottenbikes-ui:$TAG" > ui.tar

echo "--- Transferring files to $REMOTE_IP ---"
scp api.tar ui.tar "$REMOTE_USER@$REMOTE_IP:/tmp/"
scp -r internal/db/migrations "$REMOTE_USER@$REMOTE_IP:/tmp/"

echo "--- Deploying on remote server ---"
ssh "$REMOTE_USER@$REMOTE_IP" << EOF
    # Start Database quietly
    docker rm -f rottenbikes-postgres 2>/dev/null || true
    docker run -d \
      --name "rottenbikes-postgres" \
      -e POSTGRES_USER="$DB_USER" \
      -e POSTGRES_PASSWORD="$DB_PASSWORD" \
      -e POSTGRES_DB="$DB_NAME" \
      -p "5432:5432" \
      -v "rottenbikes-postgres-data:/var/lib/postgresql/data" \
      postgres:16

    # Load images
    docker load < /tmp/api.tar
    docker load < /tmp/ui.tar
    
    # Remove old containers quietly
    docker rm -f rottenbikes-api rottenbikes-ui 2>/dev/null || true
    
    # Start API on port 8082
    docker run -d --name rottenbikes-api \
        -p 8082:8080 \
        -e DB_USER="$DB_USER" \
        -e DB_PASSWORD="$DB_PASSWORD" \
        -e DB_HOST="$DB_HOST" \
        -e DB_NAME="$DB_NAME" \
        -e DB_SSLMODE="disable" \
        rottenbikes-api:$TAG

    # Start UI on port 8081
    docker run -d --name rottenbikes-ui \
        -p 8081:8081 \
        rottenbikes-ui:$TAG
EOF

echo "--- Waiting for Database to be ready ---"
sleep 5

echo "--- Running migrations from local ---"
DB_DSN="postgres://$DB_USER:$DB_PASSWORD@$REMOTE_IP:5432/$DB_NAME?sslmode=disable"
migrate -path internal/db/migrations -database "$DB_DSN" up

echo "--- Cleanup local tarballs ---"
rm api.tar ui.tar

echo "Deployment complete!"
