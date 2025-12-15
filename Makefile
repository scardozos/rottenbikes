PROJECT_NAME := rottenbikes
DB_USER      := rottenbikes
DB_PASSWORD  := rottenbikes
DB_NAME      := rottenbikes
DB_PORT      := 5432
DB_DSN       := postgres://$(DB_USER):$(DB_PASSWORD)@localhost:$(DB_PORT)/$(DB_NAME)?sslmode=disable

MIGRATIONS_DIR := internal/db/migrations

.PHONY: db-up db-migrate-up db-migrate-down db-reset run

db-up:
	@echo "Starting local PostgreSQL..."
	@.scripts/run-local-postgres.sh

db-migrate-up:
	@echo "Running migrations up..."
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" up

db-migrate-down:
	@echo "Running migrations down..."
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" down 1

db-reset:
	@echo "Resetting database (drop + re-run migrations)..."
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" drop -f
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" up

run: db-up db-migrate-up
	@echo "Starting API..."
	go run ./cmd/api
