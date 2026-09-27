# API reference

The API listens on `API_PORT` (default `8080`). Routes are registered in `cmd/api/httpserver/http.go`, which is the source of truth when this page and the code disagree.

## Conventions

- **Authentication:** send `Authorization: Bearer <api_token>`. Tokens come from confirming a magic link (see [Authentication](#authentication)) and last two months. A missing, malformed, unknown or expired token gets `401`.
- **Errors** are JSON: `{"error": "<message>"}`. That includes `405 Method Not Allowed`.
- **Status codes:** invalid input is `400` (never `500`), missing resources `404`, conflicts `409`, rate limits `429` (with a `Retry-After` header on review limits).
- **Pagination:** list endpoints accept `limit` and `offset`.
- **CORS:** only origins in `CORS_ALLOWED_ORIGINS` (plus private/loopback IPs) get CORS headers; see [configuration](configuration.md).

## Authentication

Authentication is passwordless: users get a **magic link** by email. The requesting device receives a separate **poll token** and polls `/auth/poll` until the link is confirmed (possibly on another device), then gets its API token. See [architecture](architecture.md#passwordless-authentication) for the full flow.

| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `POST` | `/auth/register` | Register a new user and email them a confirmation magic link. Returns a poll token (`magic_token`) for the requesting device. Requires `captcha_token`. A taken username returns 409; a taken email gets the same response as a new registration (the account's owner is emailed a login link instead, and the returned poll token never resolves), so the endpoint does not reveal which emails are registered. | No |
| `POST` | `/auth/request-magic-link` | Request a login magic link by `email` or `username` (max 2 per user per 24h; exceeding it returns 429). Returns a poll token (`magic_token`). Requires `captcha_token`. Unknown accounts get the same response, with a poll token that never resolves and no email. | No |
| `GET` | `/auth/confirm/{token}` | Confirm the emailed magic link and receive a Bearer token. | No |
| `GET` | `/auth/poll?token=` | Exchange the poll token for the Bearer token once the link has been confirmed (one-time; for cross-device login). `404` while not confirmed. | No |
| `GET` | `/auth/verify` | Verify the current token; returns `poster_id`, `username` and `is_admin`. | **Yes** |
| `POST` | `/auth/logout` | Revoke the current session's token (other sessions stay valid). Idempotent: returns 204 even if the token is already invalid or missing. | No |
| `DELETE` | `/auth/user` | Delete your account. By default your reviews and bikes are kept but unattributed; send `{"delete_poster_subresources": true}` to delete them too. | **Yes** |
| `GET` | `/users/me/reviews` | List your reviews (`limit`, `offset`). | **Yes** |

**Captcha:** `register` and `request-magic-link` return `403` when hCaptcha rejects the token and `503` when it could not be verified (hCaptcha unreachable or `HCAPTCHA_SECRET` missing).

**Emails** are sent in the background, so the response is the same whether or not sending succeeds; failures show up in logs and metrics ([operations](operations.md#observability)).

## Bikes

| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `GET` | `/bikes` | List bikes. Supports `q` (search on numerical/hash id), `sort` (`recent` (default), `rating`, `most_reviewed`), `limit`, `offset`. | No |
| `POST` | `/bikes` | Create a new bike (`numerical_id`: 4–5 digits, leading zeros kept; optional alphanumeric `hash_id`; `is_electric`). Accepts `was_scanned` (client-declared origin flag for moderation). Duplicate id or hash: `409`. | **Yes** |
| `GET` | `/bikes/{id}` | Get details of a specific bike. | No |
| `PUT` | `/bikes/{id}` | Update a bike's `hash_id` / `is_electric`. Only the bike's creator can update it (others get `404`). `"hash_id": ""` clears it; a hash used by another bike returns `409`. | **Yes** |
| `GET` | `/scan/{hash}` | Look up a bike by its QR `hash_id` and record the scan server-side. Reviews created afterwards get `was_scanned = true` (derived from the recorded scan, so it cannot be spoofed by clients). | **Yes** |
| `GET` | `/bikes/{id}/details` | Get bike details including windowed aggregate ratings and reviews (`limit`, `offset` apply to reviews). | No |
| `GET` | `/bikes/{id}/reviews` | List a bike's reviews (`limit`, `offset`). | No |
| `POST` | `/bikes/{id}/reviews` | Create a review for a specific bike. `was_scanned` is computed server-side from recorded scans. | **Yes** |

Bikes cannot be deleted by regular users; admins can delete any bike (see [Admin](#admin-moderation)).

## Reviews

| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `GET` | `/reviews/{id}` | Get a specific review. | No |
| `PUT` | `/reviews/{id}` | Update one of your reviews. | **Yes** |
| `DELETE` | `/reviews/{id}` | Delete one of your reviews. | **Yes** |

**Review rules:**
- Ratings: `overall`, `breaks`, `seat`, `sturdiness`, `power`, `pedals`, each optional and between 1 and 5.
- `comment` is at most 500 characters.
- A review for a missing bike returns `404`.
- Rate limits: one review per bike every 10 minutes, and at most 5 reviews per hour.
- Someone else's review returns `404` on update or delete.

## Admin (moderation)

Requires the caller to have the **admin role** (see [admin roles](operations.md#admin-roles)); others get `403`.

| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `GET` | `/admin/users?q=` | Search posters by email/username, with their role, review and bike counts (`limit`, max 50). | **Yes (admin)** |
| `DELETE` | `/admin/users/{id}` | Purge a malicious poster and all their content (reviews, ratings, created bikes, sessions) in one transaction. Deleting their bikes also removes other users' reviews on those bikes. The action is recorded in the `moderation_actions` audit table. Admins cannot purge other admins. | **Yes (admin)** |
| `DELETE` | `/admin/bikes/{id}` | Delete any bike, together with its reviews. The action is recorded in `moderation_actions`. | **Yes (admin)** |

## System

| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `GET` | `/healthz` | Liveness check. | No |
| `GET` | `/readyz` | Readiness check (pings the database). | No |

Prometheus metrics are served on a separate port; see [operations](operations.md#observability).
