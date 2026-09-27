# Architecture

## Components

| Component | Code | What it does |
| :--- | :--- | :--- |
| API | `cmd/api` | Go (`net/http`) JSON API on `:8080`, Prometheus metrics on `:9091`. |
| Web server | `cmd/web` | Serves the exported web UI (`ui/dist`) on `:8081` and injects `EXPO_PUBLIC_*` settings at runtime. |
| UI | `ui/` | React Native + Expo app for iOS, Android and web. |
| Database | `internal/db/migrations` | PostgreSQL, schema managed with `golang-migrate`. |
| `adminctl` | `cmd/adminctl` | CLI for out-of-band admin tasks (promote/demote admins, delete bikes); see [operations](operations.md#admin-roles). |

## Repository layout

```
cmd/
  api/            API entry point (main.go)
    httpserver/   HTTP handlers, routing (http.go), middleware, metrics
    email/        Email senders (Mailtrap, no-op)
  web/            Static web UI server
  adminctl/       Admin CLI
internal/
  domain/         Business logic and persistence (the "source of truth")
    sql/          Raw SQL queries, embedded with //go:embed
  db/
    migrations/   golang-migrate migrations (NNNN_name.up/down.sql)
    seeds/        Dev seed data
  dbconfig/       Database DSN from the environment (shared by api and adminctl)
e2e/              End-to-end suite (build tag e2e)
ui/               Expo app
k8s/              Kubernetes manifests (base + dev/prd overlays)
.scripts/         Helper scripts used by the Makefile
```

## Backend

Requests flow **handler → service → store**:

- **`cmd/api/httpserver`**: parses and validates requests, maps domain errors to HTTP statuses (e.g. `domain.ErrValidation` → 400, `sql.ErrNoRows` → 404), writes JSON. Middleware adds auth (`middlewareAuth`, `middlewareAdminAuth`), CORS, JSON 405s and request logging/metrics.
- **`domain.Service`** (`internal/domain/service.go`): the interface handlers depend on. It validates input (`internal/domain/validation.go`) and delegates to the store.
- **`domain.Store`**: talks to Postgres with raw SQL. There is no ORM: every query lives in `internal/domain/sql/<name>.sql` and is embedded with `//go:embed`. Multi-step operations run in a transaction.

### Data model

| Table | Holds |
| :--- | :--- |
| `posters` | Users: email, username, `email_verified`, `role` (`user`/`admin`), `is_test` (E2E test account). |
| `poster_tokens` | API sessions (SHA-256 of the token, expiry). One poster can have several. |
| `magic_links` | Pending/consumed magic links: hashed magic token, hashed poll token, expiry. |
| `bikes` | `numerical_id` (text, 4–5 digits, primary key), optional unique `hash_id` (QR code), `is_electric`, creator, `is_test` (inherited from the creator). |
| `reviews`, `review_ratings` | A review and its per-category scores (1–5). |
| `rating_aggregates` | Cached per-bike averages, recomputed whenever a review changes. |
| `scan_events` | Last QR scan per poster and bike (drives `was_scanned`). |
| `moderation_actions` | Audit log of admin purges and bike deletions. |

## How it works

### Passwordless authentication

1. The user asks for a link (`/auth/register` or `/auth/request-magic-link`), protected by [hCaptcha](https://www.hcaptcha.com/).
2. The API creates a **magic token** (emailed as a `/confirm/{token}` link to the UI) and a separate **poll token** (returned to the requesting device). Both are stored hashed.
3. Clicking the link confirms it (`/auth/confirm/{token}`): the clicking device gets an API token directly, and the poll token becomes redeemable once.
4. The requesting device, which polls `/auth/poll`, then receives the API token too. That's how "request on mobile, confirm on desktop" logs the phone in.

Details that matter:
- API tokens are only stored as SHA-256 hashes.
- The poll token can't confirm a link, and the emailed token can't poll.
- Neither endpoint reveals whether an account exists (see [API](api.md#authentication)).

### Bike scanning

The app has a built-in QR/barcode scanner. Scanning looks the bike up by `hash_id` (`GET /scan/{hash}`) and records a `scan_events` row; if the bike is unknown, the user is offered to create it. Reviews get `was_scanned = true` only if the author scanned that bike, derived server-side so clients can't spoof it. `was_scanned = false` is a moderation signal, not proof of abuse (desktop web users can't scan).

### Reviews and ratings

Reviews rate a bike on **overall**, **breaks**, **seat**, **sturdiness**, **power** (electric bikes) and **pedals**. Limits: one review per bike every 10 minutes and 5 per hour, per poster (enforced in a transaction that locks the poster row). Bike details show averages over the last week, the last two weeks and overall.

### Moderation

Admins can search posters, purge a poster with all their content, and delete any bike. Every action writes a `moderation_actions` row in the same transaction. Admins can't purge other admins. The admin role lives only in the database; see [operations](operations.md#admin-roles).

## Frontend

- **Stack:** Expo, React Native, React Navigation (native stack + bottom tabs), Axios, React Context for state.
- **`src/services/api.js`:** the Axios client. It attaches the stored Bearer token and emits `session_expired` on 401s (except from logout).
- **`src/context/`:** `AuthContext` (login, polling, logout), `ThemeContext`, `LanguageContext`, `SessionContext`, `ToastContext`.
- **`src/navigation/AppNavigator.js`:**
  - Public stack: Login and Register.
  - Signed-in tabs: Home (scanner), Bikes, My Reviews, Configuration.
  - Deep links: `rottenbikes://` and the web origin, e.g. `confirm/:token`.
- **`src/screens/`, `src/components/`:** screens and shared components.
- **`src/utils/`:** pure helpers, unit-tested.
- **`src/translations/`:** `en`, `es` and `ca` strings; see [contributing](contributing.md#ui).
- **Web and native:** the same code runs on both. Native-only modules (camera/scanner) must be guarded on web.
