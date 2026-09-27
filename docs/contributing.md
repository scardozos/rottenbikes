# Contributing

## Workflow

1. **Branch per change** off `main`, named by type: `feat/<short-name>`, `fix/<short-name>` or `bugfix/<short-name>`. Keep unrelated work (e.g. UI restyling and an API fix) on separate branches.
2. **Commits:** short, lowercase, imperative messages (`add e2e`, `fix infinite logout loop after account deletion`).
3. **Before pushing, sync with `main`:**
   ```sh
   git fetch origin
   git rebase origin/main
   ```
   Re-run the tests after the rebase.
4. **Open a PR** with this description:
   ```markdown
   ## Summary
   What changes and why, in a sentence or two.

   ## Changes
   - The notable changes, grouped by area (API, UI, docs, ...).

   ## Testing
   - What you ran, and what the new tests cover.
   ```
5. **Merge** with squash-and-merge, so `main` gets one commit per PR (`<title> (#<number>)`).
6. **Release** separately: bump the image tags in `k8s/<env>/kustomization.yaml` (`upgrade dev to vX.Y.Z`); see [operations](operations.md#deployment).

## Before opening a PR

- [ ] `make test` passes (Go + UI unit tests).
- [ ] `make lint` has no errors.
- [ ] `make e2e` passes against a local API. When touching sign-up/login, use the [test-keys setup](testing.md#full-sign-up-path-local) so nothing is skipped.
- [ ] New behaviour has tests (see below).
- [ ] Schema changes are **new** migrations; applied ones are never edited.
- [ ] Docs in `docs/` are updated if behaviour, endpoints, configuration or commands changed.

## Testing expectations

- **Every behaviour change or bug fix comes with unit tests**: a handler test for HTTP behaviour, a domain/sqlmock test for queries and transactions, and a UI test for logic in `ui/src/utils`.
- **API changes also get E2E coverage** in `e2e/`, asserting what a client sees. Tests must be safe on shared environments: create their own data with the suite's helpers (`newUser`, `newBike`, …, so it's flagged as test data and cleaned up), and never assume the database is empty.
- **Bug fixes start with a test that reproduces the bug.**
- How to run and write each kind of test: [testing](testing.md).

## Conventions

### Go

- **SQL:** lives only in `internal/domain/sql/<descriptive_name>.sql`, loaded with `//go:embed`. No SQL in Go strings, no ORM.
- **Migrations:** never edit a migration that has been applied; add `NNNN_name.up.sql` / `.down.sql`.
- **Errors:** wrap with context (`fmt.Errorf("load poster: %w", err)`); failing loudly beats failing silently.
- **Status codes:**
  - invalid client input is a `domain.ValidationError` (mapped to `400`), never a `500`;
  - map domain errors to statuses in the handler (`sql.ErrNoRows` → `404`, unique violations → `409`);
  - all errors use the `{"error": "..."}` envelope (`sendError`).
- **Auth and privacy:**
  - store tokens hashed, and never log raw tokens (request paths are redacted);
  - auth endpoints must not reveal whether an account or email exists;
  - send emails with `sendEmailAsync`.
- **Formatting:** `gofmt` (`make fmt`); `make lint` runs `go vet` and `golangci-lint`.

### UI

- **Components:** functional components with hooks; shared state in `src/context/`.
- **Styles:** `StyleSheet` via each screen's `createStyles(theme)`, with colours from the theme.
- **Text:** every user-facing string goes in `src/translations/{en,es,ca}.js`, with the same keys in all three. The `i18n-raw-text` and `translations` tests enforce this.
- **Logic:** keep it in pure modules under `src/utils/` so it can be unit-tested.
- **Web and native:** the code runs on both, so guard native-only modules (camera, secure storage) on web.
- **Formatting:** Prettier + ESLint (`npm run lint` in `ui/`).

### Docs

- **Source of truth:** the code. If docs and code disagree, fix the docs.
- **One home per topic:** see the index in the [README](../README.md#documentation). Link to it instead of duplicating it.

### Secrets

- Never commit env files, tokens or admin identities.
- Admin roles live only in the database ([operations](operations.md#admin-roles)).
