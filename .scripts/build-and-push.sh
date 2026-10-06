#!/bin/bash

# Usage: ./.scripts/build-and-push.sh [remote_db_ip] [tag] [env_file]
# Example: ./.scripts/build-and-push.sh 172.233.115.22 latest .env.dev
# A tag other than "latest" is also created as a git tag on HEAD and pushed.

# Parse arguments
MIGRATE_ONLY=false
SKIP_MIG=false
SKIP_BUILD=false

while [[ "$1" == --* ]]; do
    if [ "$1" == "--migrate-only" ]; then
        MIGRATE_ONLY=true
        shift
    elif [ "$1" == "--skip-migrations" ]; then
        SKIP_MIG=true
        shift
    elif [ "$1" == "--skip-build" ]; then
        SKIP_BUILD=true
        shift
    else
        echo "Unknown flag: $1"
        exit 1
    fi
done

REMOTE_DB_IP=$1
TAG=${2:-latest}
ENV_FILE=$3
DOCKER_REPO="monorailisland/rottenbikes"

if [ -z "$REMOTE_DB_IP" ]; then
    echo "Usage: $0 [--migrate-only] [remote_db_ip] [tag] [env_file]"
    echo "Example: $0 172.233.115.22 latest .env.dev"
    exit 1
fi

# A release tag (anything but "latest") is also tagged in git, on the commit the
# images are built from. The infra repo's `ops deploy` takes migrations from it.
GIT_TAG=false
if [ "$MIGRATE_ONLY" = false ] && [ "$SKIP_BUILD" = false ] && [ "$TAG" != "latest" ]; then
    GIT_TAG=true
    # Uncommitted changes would end up in the images but not in the tagged commit.
    if [ -n "$(git status --porcelain)" ]; then
        echo "Working tree has uncommitted changes; commit or stash them before building ${TAG}."
        exit 1
    fi
    git fetch --tags origin
    if [ $? -ne 0 ]; then echo "git fetch --tags failed"; exit 1; fi
    EXISTING=$(git rev-parse -q --verify "refs/tags/${TAG}^{commit}")
    if [ -n "$EXISTING" ] && [ "$EXISTING" != "$(git rev-parse HEAD)" ]; then
        echo "Git tag ${TAG} already exists on ${EXISTING}, not HEAD. Use a new tag."
        exit 1
    fi
fi

if [ "$MIGRATE_ONLY" = false ] && [ "$SKIP_BUILD" = false ]; then
    echo "--- Building Docker images ---"

    # Build API (generic, no env hardcoding)
    echo "Building API image..."
    docker build --target api -t "${DOCKER_REPO}:api-${TAG}" .
    if [ $? -ne 0 ]; then echo "API build failed"; exit 1; fi
    docker tag "${DOCKER_REPO}:api-${TAG}" "${DOCKER_REPO}:api-latest"

    # Build UI (generic, no env hardcoding)
    echo "Building UI image..."
    docker build --target ui -t "${DOCKER_REPO}:ui-${TAG}" .
    if [ $? -ne 0 ]; then echo "UI build failed"; exit 1; fi
    docker tag "${DOCKER_REPO}:ui-${TAG}" "${DOCKER_REPO}:ui-latest"

    echo "--- Pushing images to Docker Hub ---"
    docker push "${DOCKER_REPO}:api-${TAG}"
    if [ $? -ne 0 ]; then echo "API push failed"; exit 1; fi
    docker push "${DOCKER_REPO}:api-latest"

    docker push "${DOCKER_REPO}:ui-${TAG}"
    if [ $? -ne 0 ]; then echo "UI push failed"; exit 1; fi
    docker push "${DOCKER_REPO}:ui-latest"

    if [ "$GIT_TAG" = true ]; then
        echo "--- Tagging $(git rev-parse --short HEAD) as ${TAG} ---"
        git tag -f "${TAG}" HEAD && git push origin "refs/tags/${TAG}"
        if [ $? -ne 0 ]; then echo "Git tag push failed"; exit 1; fi
    fi
else 
    echo "--- Skipping Build and Push (Migrate Only) ---"
fi

if [ "$SKIP_MIG" = false ]; then
    echo "--- Running migrations against remote database ---"
    # Load env file if provided to get DB credentials for migration
    if [ -n "$ENV_FILE" ] && [ -f "$ENV_FILE" ]; then
        echo "Loading environment from $ENV_FILE for migrations..."
        set -o allexport
        source "$ENV_FILE"
        set +o allexport
    fi

    if [ -z "$DB_USER" ] || [ -z "$DB_PASSWORD" ] || [ -z "$DB_NAME" ]; then
        echo "Error: DB_USER, DB_PASSWORD, and DB_NAME must be set in environment or provided via env_file"
        exit 1
    fi

    # URL encode the password to safely handle special characters in the connection string
    DB_PASSWORD_ENCODED=$(python3 -c "import urllib.parse, sys; print(urllib.parse.quote(sys.argv[1]))" "$DB_PASSWORD")

    DB_DSN="postgres://$DB_USER:$DB_PASSWORD_ENCODED@$REMOTE_DB_IP:${DB_PORT:-5432}/$DB_NAME?sslmode=disable"
    echo "Migrating database at $REMOTE_DB_IP:${DB_PORT:-5432}..."
    migrate -path internal/db/migrations -database "$DB_DSN" up

    if [ $? -ne 0 ]; then 
        echo "Migration failed!"
        exit 1
    fi
    echo "Database migrations applied to $REMOTE_DB_IP"
else
    echo "--- Skipping Migrations ---"
fi
