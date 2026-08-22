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

.PHONY: db-up db-migrate-up db-migrate-down db-reset run test test-go test-ui build lint fmt build-and-push

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
	@if [ -z "$(ENV)" ]; then echo "Usage: make db-reset ENV=local/dev/prod"; exit 1; fi
	@echo "Resetting database (drop + re-run migrations)..."
	@.scripts/reset_db.sh .env.$(ENV)


db-seed:
	@echo "Seeding database..."
	@psql "$(DB_DSN)" -f internal/db/seeds/dev_seeds.sql

run: db-up db-migrate-up db-seed
	@echo "Starting API and Expo..."
	@(cd ui && npx expo start &)
	go run ./cmd/api

build:
	@echo "Building API server..."
	@go build -o bin/api ./cmd/api
	@echo "Building Web server..."
	@go build -o bin/web ./cmd/web

test:
	@echo "Running tests..."
	@HCAPTCHA_SECRET= go test ./...
	@$(MAKE) test-ui

test-go:
	@echo "Running Go tests..."
	@HCAPTCHA_SECRET= go test ./...

test-ui:
	@echo "Running UI tests..."
	@cd ui && node scripts/run-tests.js

fmt:
	@echo "Formatting code..."
	@go fmt ./...

lint:
	@echo "Linting code..."
	@go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping advanced linting."; \
	fi

build-and-push:
	@if [ -z "$(IP)" ]; then \
		echo "Usage: make build-and-push IP=<remote_db_ip> [TAG=<tag>] [ENV_FILE=<env_file>]"; \
		exit 1; \
	fi
	@./.scripts/build-and-push.sh $(IP) $(TAG) $(ENV_FILE)

