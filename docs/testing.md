# Testing

## Layers

| Layer | What | How to run |
| :--- | :--- | :--- |
| Go unit | Handlers, domain and SQL (sqlmock). Also covers hCaptcha verification (success, rejection, unreachable, bad response) and the Mailtrap sender (non-2xx, unreachable) against `httptest` servers, and the emailed `/confirm/{token}` link. | `make test-go` |
| UI unit | Pure helpers in `ui/src/utils`, translations and i18n checks. | `make test-ui` |
| E2E, local with test keys | The full sign-up path through the real API: register → email → login code, using hCaptcha's test keys (and optionally a Mailtrap sandbox inbox). | `make e2e CAPTCHA_TOKEN=...` ([below](#full-sign-up-path-local)) |
| E2E on dev / prod | Everything except passing the captcha (both use the real captcha and send real emails), plus negative probes that prove captcha is enforced (a bogus token must get 403; required on prod). | `make e2e ENV=dev`, `make e2e ENV=prod CONFIRM=prod` |
| Monitoring | Prod captcha/email failures. | Prometheus metrics ([operations](operations.md#observability)) |

`make test` runs both unit layers. The captcha widget and deep links in the UI are out of scope for the API suite; check them with Playwright or manually (a local UI with the test sitekey `10000000-ffff-ffff-ffff-000000000001` can use the test keys).

## Go unit tests

- **Domain (`internal/domain/*_test.go`):** stores are tested with [`go-sqlmock`](https://github.com/DATA-DOG/go-sqlmock), which expects each query (matched by regex) in order, including `Begin`/`Commit`/`Rollback`.
- **Handlers (`cmd/api/httpserver/*_test.go`):** tested through the real router (`srv.server.Handler.ServeHTTP`) with a `MockService` (`mock_service_test.go`), so middleware and error mapping are covered too.
- **Outbound calls:** hCaptcha and Mailtrap are pointed at `httptest` servers (`srv.captchaVerifyURL`, `MailtrapSender.APIURL`). Emails go through `recordingSender`. They are sent in the background, so call `srv.pendingEmails.Wait()` before asserting on them.
- **Environment:** `make test-go` clears `HCAPTCHA_SECRET`. Tests that need captcha behaviour set `HCAPTCHA_SECRET` / `APP_ENV` with `t.Setenv`.

## UI unit tests

- **Runner:** `ui/scripts/run-tests.js` runs every `ui/scripts/tests/*.test.js` as a plain Node process, with no Jest or bundler.
- **Helpers:** `describe`, `it`, `eq`, `ok` and `finish` come from `tests/helpers.js`. Call `finish()` at the end of the file so failures set the exit code.
- **Loading source:** `loadModule('src/utils/x.js', { mocks })` (from `scripts/load-module.js`) loads an ES module from `src/` and can stub its imports. Keep testable logic in pure modules under `src/utils/`.
- **Async tests:** `await` each `it(...)` outside `describe` (it runs its body synchronously) before calling `finish()`; see `session.test.js`.

## End-to-end tests

The `e2e/` suite exercises the real API over HTTP against a live environment:

```sh
make e2e                            # local (http://localhost:8080 + local DB)
make e2e ENV=dev                    # https://api-dev.rottenbik.es, DB from .env.dev
make e2e ENV=prod CONFIRM=prod      # https://api.rottenbik.es, DB from .env.prod
make e2e ENV=dev API_URL=http://... # override the API URL
make e2e CAPTCHA_TOKEN=...          # a captcha token the target accepts
```

- **Database access:** the suite needs the target's database (`DATABASE_URL` or the `DB_*` variables from `.env.<env>`). Tests that aren't about sign-up seed their users there, and the suite promotes test users to admin like `adminctl` does.
- **Test accounts on dev/prod:** on shared environments the suite's accounts are flagged `posters.is_test` (directly in the database, never through the API). That has three effects:
  - **Hidden:** their bikes inherit the flag and are left out of `GET /bikes` listings and search for everyone except test accounts, even after their creator is deleted.
  - **Separate numbers:** they must use the reserved 6-digit bike numbers, which real bikes (4–5 digits) can never have, so a test never takes, conflicts with, or cleans up a real bike.
  - **Safe cleanup:** cleanup only deletes bikes flagged as test.
- **Regular accounts on local:** locally the suite uses regular accounts with real 4–5 digit numbers, so the normal write path is covered end to end. The hiding and number-range rules are still tested there with explicit test accounts.
- **Overriding the mode:** `make e2e TEST_ACCOUNTS=1` rehearses the dev/prod mode locally; `TEST_ACCOUNTS=0` does the reverse, at your own risk on a shared environment.
- **Cleanup:** it deletes everything it created when it finishes. Test users are named `e2etest*`; leftover test users and test bikes from aborted runs are swept on the next run.
- **Build tag:** the suite is behind the `e2e` build tag, so `go test ./...` does not run it.
- **Runner:** `make e2e` calls `.scripts/run-e2e.sh`, which sets the `E2E_*` variables the suite reads (`E2E_API_URL`, `E2E_DB_DSN`, `E2E_CAPTCHA_TOKEN`, …). To run against a non-default port, set them yourself and call `go test -tags e2e ./e2e/...`.

**Captcha.** The suite probes the target with a bogus token. If captcha is enforced and no accepted token is available, the tests that must call `/auth/register` or `/auth/request-magic-link` are skipped. On prod, captcha *not* being enforced is a failure.

### Full sign-up path (local)

Dev and prod keep the real captcha and deliver emails to real users, so the suite can't sign up there. To cover register → email → login code, run a local API with hCaptcha's official test secret, which accepts the public test token:

```sh
HCAPTCHA_SECRET=0x0000000000000000000000000000000000000000 APP_ENV=local go run ./cmd/api
make e2e CAPTCHA_TOKEN=10000000-aaaa-bbbb-cccc-000000000001
```

The API above sends no email (no Mailtrap token), so the suite takes the magic token from the database.

To also cover email delivery:
- point the API at a Mailtrap sandbox inbox with `EMAIL_SENDER_TOKEN_MAILTRAP=<sandbox token>` and `MAILTRAP_API_URL=https://sandbox.api.mailtrap.io/api/send/<inbox_id>`;
- set `MAILTRAP_ACCOUNT_ID` and `MAILTRAP_INBOX_ID` in `.env.local`, plus `MAILTRAP_INBOX_API_TOKEN` if reading needs a different token.

The suite then reads the magic links from that inbox and deletes them.
