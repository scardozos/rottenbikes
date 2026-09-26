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
To drop the database and re-apply all migrations (fresh start):
```bash
make db-reset
```

## API Endpoints

### Authentication
| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `POST` | `/auth/request-magic-link` | Request a magic link for login. | No |
| `POST` | `/auth/register` | Register a new user. | No |
| `GET` | `/auth/confirm` | Confirm magic link (via `?token=...` or `/token`) and receive Bearer token. | No |
| `GET` | `/auth/poll` | Check status of a magic link request (for mobile polling). | No |
| `GET` | `/auth/verify` | Verify if current token is valid. | **Yes** |

### Bikes
| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `GET` | `/bikes` | List all bikes. | **Yes** |
| `POST` | `/bikes` | Create a new bike. Accepts `was_scanned` (client-declared origin flag for moderation). | **Yes** |
| `GET` | `/bikes/{id}` | Get details of a specific bike. | **Yes** |
| `PUT` | `/bikes/{id}` | Update a specific bike. | **Yes** |
| `DELETE` | `/bikes/{id}` | Delete a specific bike. | **Yes** |
| `GET` | `/scan/{hash}` | Look up a bike by its QR `hash_id` and record the scan server-side. Reviews created afterwards get `was_scanned = true` (derived from the recorded scan, so it cannot be spoofed by clients). | **Yes** |
| `GET` | `/bikes/{id}/details` | Get bike details including aggregate ratings and reviews. | **Yes** |
| `POST` | `/bikes/{id}/reviews` | Create a review for a specific bike. `was_scanned` is computed server-side from recorded scans. | **Yes** |

### Reviews
| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `GET` | `/reviews/{id}` | Get a specific review. | **Yes** |
| `PUT` | `/reviews/{id}` | Update a specific review. | **Yes** |
| `DELETE` | `/reviews/{id}` | Delete a specific review. | **Yes** |

### Admin (Moderation)
Requires the caller to have the **admin role** (see [Admin roles](#admin-roles)).

| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `GET` | `/admin/users?q=` | Search posters by email/username, with their role, review and bike counts. | **Yes (admin)** |
| `DELETE` | `/admin/users/{id}` | Purge a malicious poster and all their content (reviews, ratings, created bikes, sessions) in one transaction. Deleting their bikes also removes other users' reviews on those bikes. The action is recorded in the `moderation_actions` audit table. Admins cannot purge other admins. | **Yes (admin)** |

### System
| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `GET` | `/healthz` | Health check endpoint. | No |

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
- **Request Logging**: Structured logs for all HTTP requests.
- **Health Checks**: `/healthz` endpoint for liveness probes.

## Configuration

The application is configured via environment variables. Create a `.env` file (or set them in your environment/Docker):

| Variable | Description | Default |
| :--- | :--- | :--- |
| `DATABASE_URL` | Full Postgres connection string. | `postgres://...` (built from other vars) |
| `API_PORT` | Port for the Main API. | `8080` |
| `METRICS_PORT` | Port for Prometheus metrics. | `9091` |
| `EMAIL_SENDER_TOKEN_MAILTRAP` | API Token for Mailtrap (for sending emails). | Empty (uses No-op sender) |
| `EMAIL_FROM_ADDRESS` | Sender email address. | `hello@rottenbik.es` |
| `HCAPTCHA_SECRET` | Secret key for hCaptcha verification. | Empty (skips verification in dev) |
| `CORS_ALLOWED_ORIGINS` | Comma-separated allowlist of origins for the API (e.g. `https://rottenbik.es,https://app.rottenbik.es`). When unset, falls back to the local dev UI origins. | `http://localhost:8081,http://localhost:8080` |
| `UI_HOST` | Hostname for generating magic links. | `localhost` |
| `UI_PORT` | Port for generating magic links. | `8081` |

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

