# Rotten Bikes

An application for managing (bicing) bike reviews and ratings.

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
| `GET` | `/bikes` | List all bikes. | No |
| `POST` | `/bikes` | Create a new bike. | **Yes** |
| `GET` | `/bikes/{id}` | Get details of a specific bike. | No |
| `PUT` | `/bikes/{id}` | Update a specific bike. | **Yes** |
| `DELETE` | `/bikes/{id}` | Delete a specific bike. | **Yes** |
| `GET` | `/bikes/{id}/details` | Get bike details including aggregate ratings and reviews. | No |
| `POST` | `/bikes/{id}/reviews` | Create a review for a specific bike. | **Yes** |

### Reviews
| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `GET` | `/reviews/{id}` | Get a specific review. | No |
| `PUT` | `/reviews/{id}` | Update a specific review. | **Yes** |
| `DELETE` | `/reviews/{id}` | Delete a specific review. | **Yes** |

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
| `UI_HOST` | Hostname for generating magic links. | `localhost` |
| `UI_PORT` | Port for generating magic links. | `8081` |
