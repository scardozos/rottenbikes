# Rotten Bikes - Agent Context

Rotten Bikes is a platform for reviewing and rating shared city bikes (e.g. "Bicing"): users scan a bike's QR code, see its condition from other users' reviews, and rate it. Go API + PostgreSQL, React Native (Expo) app for iOS, Android and web.

The project docs live in `docs/` and are the source of truth for everything below. Read the relevant one before changing an area:

| Topic | Doc |
| :--- | :--- |
| Code layout, layers, data model, how auth/scanning/reviews work, frontend structure | [docs/architecture.md](docs/architecture.md) |
| Local setup, running, database tasks, recipes (endpoint, query, migration) | [docs/development.md](docs/development.md) |
| Endpoints, errors, rate limits | [docs/api.md](docs/api.md) |
| Environment variables, env files, k8s config | [docs/configuration.md](docs/configuration.md) |
| Unit and E2E tests | [docs/testing.md](docs/testing.md) |
| Deployment, admin roles, observability | [docs/operations.md](docs/operations.md) |
| Workflow, PR checklist, conventions | [docs/contributing.md](docs/contributing.md) |

## Domain language

*   **Bike**: the physical asset being reviewed, identified by its `numerical_id` (the 4–5 digit number on the frame, stored as text so leading zeros are kept) and optionally a QR `hash_id`.
*   **Poster**: a user. **Admin** is a poster with `role = 'admin'`.
*   **Review**: a rating of a bike: an overall score plus sub-scores (breaks, seat, sturdiness, power, pedals).
*   **Rating aggregate**: cached per-bike averages (`rating_aggregates`), recomputed when reviews change.
*   **Magic link**: the only login mechanism; no passwords are stored.
*   **Domain**: `internal/domain`, where business logic and database access live.
*   **Seed**: dev data in `internal/db/seeds/dev_seeds.sql`.

## How the AI Agent Should Help

**Coding Guidelines** (full list in [docs/contributing.md](docs/contributing.md#conventions)):
*   **Language**: Go (Latest/Stable), React (Functional Components + Hooks).
*   **SQL pattern**: **Do not hardcode SQL in Go strings.**
    *   Put raw SQL in `internal/domain/sql/<descriptive_name>.sql`.
    *   Use `//go:embed` to load it in the domain package.
*   **Error Handling**: Wrap errors with context in Go. Failing loudly is better than silent failure. Invalid client input is a `domain.ValidationError` (400), never a 500.
*   **Frontend**: Use `StyleSheet` in React Native. Prefer functional components. Every user-facing string goes in `ui/src/translations/{en,es,ca}.js`.
*   **Tests**: behaviour changes come with unit tests; API changes also with E2E coverage ([docs/testing.md](docs/testing.md)).
*   **Docs**: when you change behaviour, endpoints, configuration or commands, update the matching file in `docs/`. If docs and code disagree, the code is right: fix the docs.

**Do Not Touch:**
*   `internal/db/migrations/*.sql` (Old migrations): *Never* edit an existing applied migration file. Create a new one.

**Secrets:**
*   Never output real secrets. Use placeholders like `REDACTED` or reference env vars.
*   Never commit env files or admin identities (admins are a database role).

## Limitations and Known Quirks

*   **Auth Flow**: The magic link flow is complex: a request, an emailed magic token (confirm) and a separate poll token (the requesting device), both stored hashed. Auth endpoints must not reveal whether an account exists. See [docs/architecture.md](docs/architecture.md#passwordless-authentication).
*   **Expo Web vs Native**: The UI runs on both. Verify that UI changes (especially native modules like Camera/Scanner) are compatible with or guarded for Web.
*   **No ORM**: The project uses raw SQL. You must be comfortable writing and debugging PostgreSQL queries.
*   **Bikes tab**: a custom tab-press listener in `AppNavigator.js` resets the Bikes stack when the tab is tapped.
