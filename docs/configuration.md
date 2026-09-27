# Configuration

Everything is configured through environment variables. The binaries don't read `.env` files themselves: the Makefile and scripts load them (see [env files](#env-files)).

## API (`cmd/api`)

| Variable | Description | Default |
| :--- | :--- | :--- |
| `DATABASE_URL` | Full Postgres connection string. Takes precedence over the `DB_*` variables. | Built from `DB_*` |
| `DB_USER`, `DB_PASSWORD`, `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_SSLMODE` | Database connection parts (also used by `adminctl`). | `rottenbikes`, `rottenbikes`, `localhost`, `5432`, `rottenbikes`, `disable` |
| `API_PORT` | Port for the main API. | `8080` |
| `METRICS_PORT` | Port for Prometheus metrics. | `9091` |
| `APP_ENV` | `local`, `development` or `production`. | Empty |
| `HCAPTCHA_SECRET` | Secret key for hCaptcha verification. | Empty: verification is skipped when `APP_ENV` is `local`/`dev`/`development`; otherwise sign-up returns 503 |
| `EMAIL_SENDER_TOKEN_MAILTRAP` | Mailtrap API token for sending emails. | Empty: emails are not sent (no-op sender) |
| `MAILTRAP_API_URL` | Mailtrap sending endpoint. Set to `https://sandbox.api.mailtrap.io/api/send/<inbox_id>` to deliver to a sandbox inbox. | `https://send.api.mailtrap.io/api/send` |
| `EMAIL_FROM_ADDRESS` | Sender email address. | `hello@rottenbik.es` |
| `EMAIL_FROM_NAME` | Sender name. | `RottenBikes` |
| `UI_HOST`, `UI_PORT` | Host and port of the UI used to build the emailed `/confirm/{token}` links (`http` for localhost/private IPs, `https` otherwise). | `localhost`, `8081` |
| `CORS_ALLOWED_ORIGINS` | Comma-separated allowlist of origins (e.g. `https://rottenbik.es`). Private/loopback IP origins are always allowed. | `http://localhost:8081,http://localhost:8080` |

## Web server (`cmd/web`) and UI

| Variable | Where | Description |
| :--- | :--- | :--- |
| `PORT` | `cmd/web` | Port to serve `ui/dist` on (default `8081`). |
| `ENV` | `cmd/web` | `local` or empty enables human-readable logs. |
| `EXPO_PUBLIC_API_URL` | `cmd/web` (runtime), `ui/.env.local` (dev), `ui/eas.json` (mobile builds) | API base URL. `cmd/web` injects it into the page at runtime, so one UI image works for every environment. |
| `EXPO_PUBLIC_HCAPTCHA_SITEKEY` | same as above | hCaptcha sitekey matching the API's `HCAPTCHA_SECRET`. |

## Env files

| File | Used by | Contains |
| :--- | :--- | :--- |
| `.env.local` | `Makefile` (loaded and exported for every target), `make e2e` | Local API settings (captcha, Mailtrap, UI host). |
| `.env.dev`, `.env.prod` | `.scripts/*` (`db-reset`, `reset-login-*`, `create_k8s_secrets.sh`, `run-e2e.sh`), `admin-*` targets via `ENV_FILE=` | Remote DB credentials and secrets for that environment. |
| `ui/.env.local` | Expo | `EXPO_PUBLIC_*` for local development. |

Env files contain secrets and are git-ignored; never commit them. `ADMIN_EMAILS` (still present in some env files) is no longer read: admins are a database role (see [operations](operations.md#admin-roles)).

## Kubernetes

- **Non-secret settings** live in the ConfigMap: `k8s/base/configmap.yaml`, merged with per-environment values in `k8s/{dev,prd}/kustomization.yaml`.
- **Secrets** are created from the env files by `.scripts/create_k8s_secrets.sh`:
  - required: `DB_USER`, `DB_PASSWORD`, `EMAIL_SENDER_TOKEN_MAILTRAP`;
  - optional, included when set: `HCAPTCHA_SECRET`, `MAILTRAP_API_URL`.

See [operations](operations.md#deployment).
