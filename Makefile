PROJECT_NAME := rottenbikes
DB_USER      := rottenbikes
DB_PASSWORD  := rottenbikes
DB_NAME      := rottenbikes
DB_PORT      := 5432
DB_DSN       := postgres://$(DB_USER):$(DB_PASSWORD)@localhost:$(DB_PORT)/$(DB_NAME)?sslmode=disable

ifneq (,$(wildcard ./.env.local))
    include .env.local
    export
endif

MIGRATIONS_DIR := internal/db/migrations

.PHONY: db-up db-migrate-up db-migrate-down db-reset run

db-up:
	@echo "Starting local PostgreSQL..."
	@.scripts/run-local-postgres.sh
	@sleep 2

db-migrate-up:
	@echo "Running migrations up..."
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" up

db-migrate-down:
	@echo "Running migrations down..."
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" down 1

reset-login-local:
	@if [ -z "$(USER)" ]; then echo "Usage: make reset-login-local USER=<email_or_username>"; exit 1; fi
	@.scripts/reset_login_attempts.sh $(USER) .env.local

reset-login-dev:
	@if [ -z "$(USER)" ]; then echo "Usage: make reset-login-dev USER=<email_or_username>"; exit 1; fi
	@.scripts/reset_login_attempts.sh $(USER) .env.dev

reset-login-prd:
	@if [ -z "$(USER)" ]; then echo "Usage: make reset-login-prd USER=<email_or_username>"; exit 1; fi
	@.scripts/reset_login_attempts.sh $(USER) .env.prod

db-reset:

	@echo "Resetting database (drop + re-run migrations)..."
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" drop -f
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" up


db-seed:
	@echo "Seeding database..."
	@psql "$(DB_DSN)" -f internal/db/seeds/dev_seeds.sql

run: db-up db-migrate-up db-seed
	@echo "Starting API and Expo..."
	@(cd ui && npx expo start &)
	go run ./cmd/api

