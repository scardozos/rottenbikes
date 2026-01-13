# Rotten Bikes - Project Context

## Project Overview

Rotten Bikes is a platform for reviewing and rating shared city bikes (e.g., "Bicing"). It allows users to scan a bike's QR code, view its maintenance history defined by user reviews, and submit their own ratings on specific attributes like breaks, seat comfort, and sturdiness.

**Key Technologies:**
*   **Backend:** Go (1.25+), Standard `net/http`, `lib/pq`.
*   **Frontend:** React Native (Expo) for Mobile (iOS/Android) and Web.
*   **Database:** PostgreSQL.
*   **Infrastructure:** Kubernetes (k8s), Docker.
*   **Tools:** `make`, `golang-migrate` (DB migrations).

## Core Concepts and Domain Language

*   **Bike**: The physical asset being reviewed. Identified by a `numerical_id` (the visible number on the bike frame) and an internal UUID.
*   **Review**: A structured rating submitted by a user for a specific bike. Includes an overall score and sub-scores (Breaks, Seat, etc.).
*   **Magic Link**: The primary authentication mechanism. Users request a link via email; no passwords are stored.
*   **Rating Aggregate**: A denormalized or calculated summary of a bike's performance (e.g., "Average Brake Rating").
*   **Domain**: Refers to `internal/domain`, where business logic and database interactions reside.
*   **Seed**: Refers to initial data population (handled by `internal/db/seeds/dev_seeds.sql`).

## Repository Structure

*   **`cmd/api/`**: Main entry point for the Backend API.
    *   `main.go`: Application bootstrapping.
    *   `httpserver/`: HTTP handlers and routing logic.
*   **`internal/domain/`**: The core business logic "Source of Truth".
    *   Contains types (`bike.go`, `review.go`), service logic, and persistence.
    *   `sql/`: Raw SQL queries live here (embedded into Go binaries).
*   **`internal/db/`**: Database artifacts.
    *   `migrations/`: SQL migration files (Golang Migrate format).
    *   `seeds/`: Data seeding scripts.
*   **`ui/`**: The Frontend application (Expo/React Native).
    *   `App.js` & `src/`: Source code.
    *   Running `make run` starts this alongside the backend.
*   **`k8s/`**: Kubernetes manifests.
    *   Organized by overlay: `base`, `dev`, `prd`.
*   **`Makefile`**: The control center for local development (`db-up`, `run`, `db-reset`).
*   **`.scripts/`**: Helper shell scripts used by the Makefile.

## How to Run and Develop Locally

**Standard Workflow:**
1.  **Prerequisites**: Docker, Go 1.23+, Node/npm.
2.  **Start All**: Run `make run`.
    *   This spins up Postgres (local Docker), runs migrations, seeds data, starts the Go backend (:8080), and the Expo UI (:8081).
3.  **Database Management**:
    *   `make db-reset`: Wipes the database and re-applies everything (destructive).
    *   `make db-migrate-up`: Applies pending migrations.
4.  **Verification**:
    *   Backend: `http://localhost:8080/healthz`
    *   Frontend: `http://localhost:8081`

**Configuration:**
*   Environment variables are loaded from `.env` files (e.g., `.env.local`).
*   Example defaults are often found in `Makefile` assignments or `.env.example` if available.

## Testing, Linting, and Quality Gates

*   **Go Tests**: Standard `go test ./...` in the root.
    *   Tests often reside next to the code (e.g., `internal/domain/bike_test.go`).
    *   Uses `go-sqlmock` for database mocking in unit tests.
*   **Linting**: Standard Go formatting (`gofmt`) is expected.
*   **CI**: No active GitHub Actions workflows detected currently.
*   **Frontend**: `npm start` in `ui/` runs Expo.

## Deployment and Operations

*   **Kubernetes**: The app is deployed to K8s using manifests in `k8s/`.
    *   Structure suggests a Kustomize-like approach (`base`, `dev`, `prd`).
*   **Observability**:
    *   Prometheus metrics exposed at `:9091/metrics`.
    *   Structured logging to stdout.
*   **Environments**:
    *   **Dev**: Uses `k8s/dev` and `api-dev` images.
    *   **Prd**: Uses `k8s/prd`.

## How the AI Agent Should Help

**Coding Guidelines:**
*   **Language**: Go (Latest/Stable), React (Functional Components + Hooks).
*   **SQL pattern**: **Do not hardcode SQL in Go strings.**
    *   Put raw SQL in `internal/domain/sql/<descriptive_name>.sql`.
    *   Use `//go:embed` to load it in the domain package.
*   **Error Handling**: Wrap errors with context in Go. failing loudly is better than silent failure.
*   **Frontend**: Use `StyleSheet` in React Native. prefer Functional components.

**Do Not Touch:**
*   `internal/db/migrations/*.sql` (Old migrations): *Never* edit an existing applied migration file. Create a new one.

**Secrets:**
*   Never output real secrets. Use placeholders like `REDACTED` or reference env vars.

## Common Tasks and Recipes

**Adding a Repository Method:**
1.  Write the SQL query in `internal/domain/sql/new_query.sql`.
2.  Embed it in the relevant Go file (`bike.go`, etc.).
3.  Add the method to the Service struct.
4.  Add a test case in `_test.go` using `sqlmock`.

**Adding a New API Endpoint:**
1.  Define the handler in `cmd/api/httpserver/<resource>.go`.
2.  Register the route in `cmd/api/main.go` or the router setup.
3.  Implement the business logic in `internal/domain/`.
4.  Ensure `Auth Required` logic is applied if it's protected.

**Modifying DB Schema:**
1.  Create two files in `internal/db/migrations/`: `XXXX_name.up.sql` and `XXXX_name.down.sql`.
2.  Run `make db-reset` (local) to apply and verify.

## Limitations and Known Quirks

*   **Auth Flow**: The "Magic Link" flow is complex. It involves an initial request, email delivery (mocked/Mailtrap locally), and a "verify" step. Tokens must be handled carefully.
*   **Expo Web vs Native**: The UI runs on both. Verify that UI changes (especially native modules like Camera/Scanner) are compatible with or guarded for Web.
*   **No ORM**: The project uses raw SQL. You must be comfortable writing and debugging PostgreSQL queries.
