#!/usr/bin/env bash
set -euo pipefail

CONTAINER_NAME="rottenbikes-postgres"
POSTGRES_USER="rottenbikes"
POSTGRES_PASSWORD="rottenbikes"
POSTGRES_DB="rottenbikes"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"

if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}\$"; then
  echo "Container ${CONTAINER_NAME} already exists. Starting it..."
  docker start "${CONTAINER_NAME}" >/dev/null
else
  echo "Starting new PostgreSQL container '${CONTAINER_NAME}' on port ${POSTGRES_PORT}..."
  docker run -d \
    --name "${CONTAINER_NAME}" \
    -e POSTGRES_USER="${POSTGRES_USER}" \
    -e POSTGRES_PASSWORD="${POSTGRES_PASSWORD}" \
    -e POSTGRES_DB="${POSTGRES_DB}" \
    -p "${POSTGRES_PORT}:5432" \
    -v "${CONTAINER_NAME}-data:/var/lib/postgresql/data" \
    postgres:16
fi

echo "PostgreSQL is starting."
echo "DSN: postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable"
