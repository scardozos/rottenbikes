# Development

## Prerequisites

- **Go** 1.25+ (see `go.mod`)
- **Docker** (local PostgreSQL container)
- **Node.js / npm** (UI)
- **golang-migrate** CLI (`migrate`)
- **psql** / **pg_dump** and **python3** (used by the helper scripts)
- **Make**

## Running locally

```sh
make run
```

This starts the whole stack:
1. `db-up`: a PostgreSQL container named `rottenbikes-postgres` on port 5432.
2. `db-migrate-up`: applies the migrations.
3. `db-seed`: loads `internal/db/seeds/dev_seeds.sql`. It seeds posters `alice` (admin), `bob` and `carol`, plus a few bikes and reviews.
4. Expo, in the background.
5. The API on `http://localhost:8080`.

The Makefile loads `.env.local` automatically; see [configuration](configuration.md). Checks: `http://localhost:8080/healthz` for the API, `http://localhost:8081` for the UI.

The steps can also be run on their own: `make db-up`, `make db-migrate-up`, `make db-seed`, then `go run ./cmd/api`.

## Running the UI

The UI is built with React Native and Expo, supporting mobile (iOS/Android) and web.

- **Mobile (Expo Go):** `make run` starts the Expo server; follow its instructions to open the app in Expo Go on a device or emulator. The app needs to reach the API, so set `EXPO_PUBLIC_API_URL` in `ui/.env.local` to your machine's LAN address.
- **Web, development mode (hot reload):** started by `make run` at `http://localhost:8081`.
- **Web, production mode (served by Go):**
  ```sh
  cd ui && npx expo export --platform web   # writes ui/dist
  go run ./cmd/web                          # serves ui/dist on :8081 (PORT to change it)
  ```

## Database

- **Reset:** `make db-reset ENV=local` (or `dev` / `prod`) drops the schema and re-applies all migrations, keeping the data: it is backed up to `backup_data_<env>.sql` and restored afterwards.
- **Roll back one migration:** `make db-migrate-down`.
- **Unlock a user who hit the magic-link limit:** `make reset-login-local USER=<email_or_username>` (also `reset-login-dev`, `reset-login-prd`).

## Common tasks

### Add a database change
1. Create `internal/db/migrations/NNNN_name.up.sql` and `NNNN_name.down.sql` with the next number.
2. Never edit a migration that has already been applied anywhere; add a new one instead.
3. Apply it with `make db-migrate-up` (or `make db-reset ENV=local`).

### Add a query / repository method
1. Write the SQL in `internal/domain/sql/<descriptive_name>.sql`.
2. Embed it with `//go:embed` in the relevant domain file (`bike.go`, `review.go`, …).
3. Add the method to the `Store` (and to the `Service` interface if handlers need it).
4. Test it with `sqlmock` in the matching `_test.go`.

### Add an API endpoint
1. Write the handler in `cmd/api/httpserver/<resource>.go`.
2. Register the route in `cmd/api/httpserver/http.go`, wrapped in `middlewareAuth` / `middlewareAdminAuth` if it needs a session.
3. Put the business logic in `internal/domain/`.
4. Add handler tests (with `MockService`), E2E coverage, and document it in [api.md](api.md).

## Useful Make targets

| Target | What it does |
| :--- | :--- |
| `make run` | DB + migrations + seeds + Expo + API. |
| `make test` / `test-go` / `test-ui` | Unit tests (Go + UI) / Go only / UI only. See [testing](testing.md). |
| `make e2e ENV=…` | End-to-end suite against a running environment. |
| `make fmt` | `go fmt ./...` |
| `make lint` | `go vet`, `golangci-lint` (if installed) and the UI's ESLint. |
| `make build` | Builds `bin/api` and `bin/web`. |
| `make admin-*` | Admin role management; see [operations](operations.md#admin-roles). |
| `make build-and-push` | Builds and pushes images and migrates a remote DB; see [operations](operations.md#deployment). |
