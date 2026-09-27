# Operations

## Environments

| Environment | UI | API | Namespace | Notes |
| :--- | :--- | :--- | :--- | :--- |
| local | `http://localhost:8081` | `http://localhost:8080` | – | `make run`; see [development](development.md). |
| dev | `https://dev.rottenbik.es` | `https://api-dev.rottenbik.es` | `rottenbikes-dev` | Real users: real hCaptcha and real emails. |
| prod | `https://rottenbik.es` | `https://api.rottenbik.es` | `rottenbikes-prd` | |

## Deployment

The app runs on Kubernetes, with manifests in `k8s/`:
- `k8s/base/`: deployments, services, ConfigMap.
- `k8s/dev/`, `k8s/prd/`: Kustomize overlays. They add a name suffix (`api-dev`, `api-prd`, …), the namespace, the gateway/routes/certificates, environment-specific ConfigMap values and the image tags.

Releasing a new version:

1. **Build, push and migrate:**
   ```sh
   make build-and-push IP=<db_ip> TAG=<tag> ENV_FILE=.env.dev
   ```
   Or call `.scripts/build-and-push.sh` directly to use `--migrate-only`, `--skip-migrations` or `--skip-build`. It:
   - builds the `api` and `ui` targets of the `Dockerfile`;
   - pushes them to Docker Hub as `monorailisland/rottenbikes:api-<tag>` / `ui-<tag>`, plus the matching `-latest` tags;
   - runs the migrations against the remote database.

   Migrations must be applied **before** the new API rolls out.
2. **Update secrets** if needed: `.scripts/create_k8s_secrets.sh` builds `rottenbikes-secrets-dev` / `-prd` from `.env.dev` / `.env.prod` (see [configuration](configuration.md#kubernetes)).
3. **Bump the image tags** (`newTag`) in `k8s/<env>/kustomization.yaml`, then apply the overlay, e.g. `kubectl apply -k k8s/dev`.
4. **Verify:** check `/healthz` and `/readyz`, then run the E2E suite against the environment (`make e2e ENV=dev`; see [testing](testing.md)).

## Admin roles

Admin (moderation) access is a database role, `posters.role` (`'user'` | `'admin'`), managed out-of-band, so **no admin identity ever lives in git, env files or k8s manifests**. The role is read from the database on every request, so promotion and demotion take effect immediately, without a restart or redeploy.

Manage admins with the `adminctl` CLI (or the Make wrappers). It uses the same database configuration as the API (`DATABASE_URL` or the `DB_*` variables). Pass `ENV_FILE` to target a remote environment:

```sh
make admin-promote USER=alice@example.com                     # local DB
make admin-promote USER=alice@example.com ENV_FILE=.env.prod  # prod DB
make admin-demote  USER=alice@example.com [ENV_FILE=...]
make admin-list [ENV_FILE=...]
make admin-delete-bike BIKE=1234 [ENV_FILE=...]              # not audited: no admin to attribute it to
# or directly: go run ./cmd/adminctl promote|demote|list|delete-bike <arg>
```

Notes:
- **Admins can't purge other admins.** Demote them first, so one compromised admin can't wipe the rest.
- **Audit log:** purging a poster deletes all their content and writes a row to `moderation_actions`, as does deleting a bike through the API.
- **Seed data:** the seeded dev database promotes `alice` as the dev admin.

## Observability

- **Metrics:** Prometheus, on `METRICS_PORT` (default `9091`) at `/metrics`. This port is not exposed publicly.
  - `http_requests_total{method,path,status}`, `http_request_duration_seconds`, `http_response_size_bytes`, `http_requests_in_flight`: generic HTTP metrics.
  - `captcha_verifications_total{result}`. Values:
    - `success`;
    - `failure`: hCaptcha rejected the token;
    - `error`: hCaptcha unreachable or bad response;
    - `not_configured`: no `HCAPTCHA_SECRET` outside dev;
    - `skipped`: dev without a secret.
  - `emails_sent_total{sender,kind,result}`: `kind` is `register`, `magic_link` or `existing_account`, and `result` is `success` or `failure`. Emails are sent in the background, so a failed send only shows up here and in the logs, never in the HTTP response.
- **Logs:** structured JSON (zerolog), one line per request with status, duration and username; secrets such as magic-link tokens are redacted. Lines to watch: `failed to send registration email`, `failed to send magic link email`, `failed to send existing account email`, `hCaptcha request error`.
- **Health checks:** `/healthz` (liveness) and `/readyz` (readiness, pings the database), used by the Kubernetes probes.

Prod can't be tested with a real captcha, so sign-up and login problems there (an expired Mailtrap token, a rotated hCaptcha secret, an hCaptcha outage) show up in `captcha_verifications_total`, `emails_sent_total` and `5xx` rates on `/auth/*`.
