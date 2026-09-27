# Rotten Bikes

An open-source platform for community reviews, ratings, and condition tracking of urban bike-share fleets.

> **Disclaimer:** Rotten Bikes is an independent, community-driven open-source project. It is not affiliated, associated, authorized, endorsed by, or in any way officially connected with Bicing, Barcelona de Serveis Municipals (B:SM), Smou, the Ajuntament de Barcelona, or any official public transit operator. 
>
> All product names, trademarks, and registered trademarks mentioned herein (such as "Bicing") are the property of their respective owners. Any reference to third-party services or marks is strictly for identification, reference, and descriptive nominative fair use purposes.

## Overview

Rotten Bikes allows commuters and riders to crowdsource fleet quality data for public and shared bicycles (such as Barcelona's Bicing system). Users can scan vehicle QR/Barcodes to leave and view ratings across key physical components (brakes, pedals, electric power, seat comfort) to help fellow riders avoid faulty equipment and report maintenance needs.

## Getting Started

### Prerequisites

*   **Go**: Version 1.23 or higher.
*   **PostgreSQL**: A running PostgreSQL instance.
*   **Golang Migrate**: CLI tool for database migrations.
*   **Make**: For running Makefile commands.

### Installation & Run

1.  **Start the local database:**
    This command uses a helper script to start a Postgres container (requires Docker).
    ```bash
    make db-up
    ```

2.  **Run database migrations:**
    Apply the schema to the database.
    ```bash
    make db-migrate-up
    ```

3.  **Start the API server:**
    ```bash
    make run
    ```
    This command starts the Backend API on `localhost:8080` AND the Expo development server for the UI.

## Running the UI

The UI is built with React Native and Expo, supporting both mobile (iOS/Android) and web.

### Mobile Development (Expo Go)
The `make run` command handles starting the Expo server. Follow the terminal instructions to open the app in **Expo Go** on your physical device or an emulator.

### Web UI
There are two ways to run the UI in a browser:

#### 1. Development Mode (Hot Reloading)
This is started automatically by `make run`. You can access it at `http://localhost:8081`.

#### 2. Production Mode (Served by Go)
For a production-like environment, the UI can be built and served by a dedicated Go web server:

1.  **Build the UI:**
    ```bash
    cd ui
    npx expo export --platform web
    ```
    *Note: This will generate the static files in `ui/dist` (or `ui/web-build` depending on configuration, ensure it matches `./ui/dist` for the Go server).*

2.  **Start the Go Web Server:**
    ```bash
    go run ./cmd/web
    ```
    The Web UI will be available at `http://localhost:8081` (default port).

### Database Reset
To drop the schema and re-apply all migrations, keeping the existing data (it is backed up to `backup_data_<env>.sql` and restored afterwards):
```bash
make db-reset ENV=local   # or dev / prod
```

## API Endpoints

### Authentication
| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `POST` | `/auth/register` | Register a new user and email them a confirmation magic link. Returns a poll token (`magic_token`) for the requesting device. Requires `captcha_token`. A taken username returns 409; a taken email gets the same response as a new registration (the account's owner is emailed a login link instead, and the returned poll token never resolves), so the endpoint does not reveal which emails are registered. | No |
| `POST` | `/auth/request-magic-link` | Request a login magic link by `email` or `username` (max 2 per user per 24h; exceeding it returns 429). Returns a poll token (`magic_token`). Requires `captcha_token`. Unknown accounts get the same response, with a poll token that never resolves and no email. | No |
| `GET` | `/auth/confirm/{token}` | Confirm the emailed magic link and receive a Bearer token. | No |
| `GET` | `/auth/poll?token=` | Exchange the poll token for the Bearer token once the link has been confirmed (one-time; for cross-device login). | No |
| `GET` | `/auth/verify` | Verify the current token; returns `poster_id`, `username` and `is_admin`. | **Yes** |
| `POST` | `/auth/logout` | Revoke the current session's token (other sessions stay valid). Idempotent: returns 204 even if the token is already invalid or missing. | No |
| `DELETE` | `/auth/user` | Delete your account. By default your reviews and bikes are kept but unattributed; send `{"delete_poster_subresources": true}` to delete them too. | **Yes** |
| `GET` | `/users/me/reviews` | List your reviews (`limit`, `offset`). | **Yes** |

### Bikes
| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `GET` | `/bikes` | List bikes. Supports `q` (search on numerical/hash id), `sort` (`recent` (default), `rating`, `most_reviewed`), `limit`, `offset`. | No |
| `POST` | `/bikes` | Create a new bike. Accepts `was_scanned` (client-declared origin flag for moderation). | **Yes** |
| `GET` | `/bikes/{id}` | Get details of a specific bike. | No |
| `PUT` | `/bikes/{id}` | Update a bike's `hash_id` / `is_electric`. Only the bike's creator can update it. | **Yes** |
| `GET` | `/scan/{hash}` | Look up a bike by its QR `hash_id` and record the scan server-side. Reviews created afterwards get `was_scanned = true` (derived from the recorded scan, so it cannot be spoofed by clients). | **Yes** |
| `GET` | `/bikes/{id}/details` | Get bike details including windowed aggregate ratings and reviews (`limit`, `offset` apply to reviews). | No |
| `GET` | `/bikes/{id}/reviews` | List a bike's reviews (`limit`, `offset`). | No |
| `POST` | `/bikes/{id}/reviews` | Create a review for a specific bike. `was_scanned` is computed server-side from recorded scans. | **Yes** |

Bikes cannot be deleted by regular users; admins can delete any bike (see below).

### Reviews
| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `GET` | `/reviews/{id}` | Get a specific review. | No |
| `PUT` | `/reviews/{id}` | Update one of your reviews. | **Yes** |
| `DELETE` | `/reviews/{id}` | Delete one of your reviews. | **Yes** |

### Admin (Moderation)
Requires the caller to have the **admin role** (see [Admin roles](#admin-roles)).

| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `GET` | `/admin/users?q=` | Search posters by email/username, with their role, review and bike counts (`limit`, max 50). | **Yes (admin)** |
| `DELETE` | `/admin/users/{id}` | Purge a malicious poster and all their content (reviews, ratings, created bikes, sessions) in one transaction. Deleting their bikes also removes other users' reviews on those bikes. The action is recorded in the `moderation_actions` audit table. Admins cannot purge other admins. | **Yes (admin)** |
| `DELETE` | `/admin/bikes/{id}` | Delete any bike, together with its reviews. The action is recorded in `moderation_actions`. | **Yes (admin)** |

### System
| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `GET` | `/healthz` | Liveness check. | No |
| `GET` | `/readyz` | Readiness check (pings the database). | No |

## Key Features

### 🔐 Passwordless Authentication
Rotten Bikes uses a **magic link** system for authentication, removing the need for user passwords.
- Users request a login link via email.
- The system supports seamless cross-device login: request on mobile, confirm on desktop, and the mobile app usually automatically logs in via polling.
- Protected by [hCaptcha](https://www.hcaptcha.com/) to prevent spam.

### 🚲 Bike Scanning
The mobile app features a built-in **QR/Barcode scanner**.
- Scan a bike's QR code to instantly view its details and reviews.
- If the bike doesn't exist in the system, you'll be prompted to create it immediately.
- Scans are recorded server-side (`scan_events`): reviews submitted for a scanned bike are flagged `was_scanned = true` for moderation. Reviews submitted after manual ID entry are flagged `was_scanned = false` (a moderation signal, not proof of abuse — e.g. desktop web users cannot scan).

### 📊 Review System
Rate bikes across multiple categories:
- **Overall Rating**
- **Breaks**
- **Seat Comfort**
- **Sturdiness**
- **Power** (for electric bikes)
- **Pedals**

Includes a "frequency limit" preventing users from reviewing the same bike more than once every 10 minutes.

### 🔭 Observability
The API comes with built-in instrumentation:
- **Prometheus Metrics**: Available on port `9091` at `/metrics`.
  - `http_requests_total{method,path,status}`, `http_request_duration_seconds`, … — generic HTTP metrics.
  - `captcha_verifications_total{result}` — `success`, `failure` (hCaptcha rejected the token), `error` (hCaptcha unreachable or bad response), `not_configured` (no `HCAPTCHA_SECRET` outside dev), `skipped` (dev without a secret).
  - `emails_sent_total{sender,kind,result}` — `kind` is `register`, `magic_link` or `existing_account`, `result` is `success` or `failure`. Emails are sent in the background (so response times don't reveal whether an account exists), so a failed send shows up here and in the logs, not in the HTTP response.
- **Request Logging**: Structured logs for all HTTP requests.
- **Health Checks**: `/healthz` (liveness) and `/readyz` (readiness, pings the database).

`/auth/register` and `/auth/request-magic-link` return **403** when hCaptcha rejects the token and **503** when it could not be verified (hCaptcha unreachable or `HCAPTCHA_SECRET` missing), so the two can be told apart.

## Configuration

The application is configured via environment variables. Create a `.env` file (or set them in your environment/Docker):

| Variable | Description | Default |
| :--- | :--- | :--- |
| `DATABASE_URL` | Full Postgres connection string. | `postgres://...` (built from other vars) |
| `API_PORT` | Port for the Main API. | `8080` |
| `METRICS_PORT` | Port for Prometheus metrics. | `9091` |
| `EMAIL_SENDER_TOKEN_MAILTRAP` | API Token for Mailtrap (for sending emails). | Empty (uses No-op sender) |
| `MAILTRAP_API_URL` | Mailtrap sending endpoint. Set to `https://sandbox.api.mailtrap.io/api/send/<inbox_id>` to deliver to a sandbox inbox (dev). | `https://send.api.mailtrap.io/api/send` |
| `EMAIL_FROM_ADDRESS` | Sender email address. | `hello@rottenbik.es` |
| `HCAPTCHA_SECRET` | Secret key for hCaptcha verification. | Empty (skips verification when `APP_ENV` is `local`/`dev`/`development`; otherwise sign-up returns 503) |
| `APP_ENV` | `local`, `development` or `production`. | Empty |
| `CORS_ALLOWED_ORIGINS` | Comma-separated allowlist of origins for the API (e.g. `https://rottenbik.es,https://app.rottenbik.es`). When unset, falls back to the local dev UI origins. | `http://localhost:8081,http://localhost:8080` |
| `UI_HOST` | Hostname for generating magic links. | `localhost` |
| `UI_PORT` | Port for generating magic links. | `8081` |

## Testing

Tests are layered by environment:

| Layer | What | Where |
| :--- | :--- | :--- |
| Unit | Handlers, domain and SQL (sqlmock); hCaptcha verification (success, rejection, unreachable, bad response) and the Mailtrap sender (non-2xx, unreachable) against `httptest` servers; the emailed `/confirm/{token}` link. | `make test-go` |
| E2E, local with test keys | The full sign-up path through the real API: register → email → confirm → poll, using hCaptcha's test keys (and optionally a Mailtrap sandbox inbox). | `make e2e CAPTCHA_TOKEN=...` (see below) |
| E2E on dev / prod | Everything except passing the captcha (both use the real captcha and send real emails), plus negative probes that prove captcha is enforced (a bogus token must get 403; required on prod). | `make e2e ENV=dev`, `make e2e ENV=prod CONFIRM=prod` |
| Monitoring | Prod captcha/email failures (see the metrics under Observability). | Prometheus |

The captcha widget and deep links in the UI are out of scope for the API suite; check them with Playwright or manually (a local UI with the test sitekey `10000000-ffff-ffff-ffff-000000000001` can use the test keys).

### End-to-end tests

The `e2e/` suite exercises the real API over HTTP against a live environment:

```sh
make e2e                            # local (http://localhost:8080 + local DB)
make e2e ENV=dev                    # https://api-dev.rottenbik.es, DB from .env.dev
make e2e ENV=prod CONFIRM=prod      # https://api.rottenbik.es, DB from .env.prod
make e2e ENV=dev API_URL=http://... # override the API URL
make e2e CAPTCHA_TOKEN=...          # a captcha token the target accepts
```

The suite needs access to the target's database (`DATABASE_URL` or the `DB_*` variables from `.env.<env>`). Tests that aren't about sign-up get their users seeded there, the suite promotes test users to admin like `adminctl` does, and it deletes everything it created when it finishes (test users are named `e2etest*`; leftovers of aborted runs are swept on the next run). The suite is behind the `e2e` build tag, so `go test ./...` does not run it.

**Captcha.** The suite probes the target with a bogus token. If captcha is enforced and no accepted token is available, the tests that must call `/auth/register` or `/auth/request-magic-link` are skipped; on prod, captcha *not* being enforced is a failure.

**Full sign-up path (local).** Dev and prod keep the real captcha and deliver emails to real users, so the suite cannot sign up there. To cover register → email → confirm → poll, run a local API with hCaptcha's official test secret, which accepts the public test token:

```sh
HCAPTCHA_SECRET=0x0000000000000000000000000000000000000000 APP_ENV=local go run ./cmd/api
make e2e CAPTCHA_TOKEN=10000000-aaaa-bbbb-cccc-000000000001
```

The API above sends no email (no Mailtrap token), so the suite takes the magic token from the database. To also cover email delivery, point the API at a Mailtrap sandbox inbox (`EMAIL_SENDER_TOKEN_MAILTRAP=<sandbox token>` and `MAILTRAP_API_URL=https://sandbox.api.mailtrap.io/api/send/<inbox_id>`) and set `MAILTRAP_ACCOUNT_ID` and `MAILTRAP_INBOX_ID` in `.env.local` (plus `MAILTRAP_INBOX_API_TOKEN` if reading needs a different token): the suite then reads the magic links from that inbox and deletes them.

## Admin roles

Admin (moderation) access is a database role — `posters.role` (`'user'` | `'admin'`) — managed out-of-band, so **no admin identity ever lives in git, env files, or k8s manifests**. The role is read from the database on every token lookup, so promotion and demotion take effect immediately, without a restart or redeploy.

Manage admins with the `adminctl` CLI (or the Make wrappers), which uses the same database configuration as the API (`DATABASE_URL` or the `DB_*` variables):

```sh
make admin-promote USER=alice@example.com   # or: go run ./cmd/adminctl promote alice@example.com
make admin-demote  USER=alice@example.com
make admin-list
```

In Kubernetes, run it inside an API pod so nothing sensitive touches a manifest:

```sh
kubectl exec -n rottenbikes deploy/api -- ./adminctl list
```

Notes:
- Admins cannot purge other admins (demote first) — one compromised admin can't wipe the rest.
- Purging a poster deletes all their content and writes an audit row to `moderation_actions`.
- The seeded dev database promotes `alice` as the dev admin.

