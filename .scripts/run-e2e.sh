#!/usr/bin/env bash
# Runs the E2E suite (e2e/) against a live environment.
#
# Usage: .scripts/run-e2e.sh <local|dev|prod> [extra go test args...]
#
# The API URL defaults per environment and can be overridden with API_URL.
# Database settings come from .env.<env> (DATABASE_URL or DB_*), like the
# other scripts; the suite needs DB access to seed test users (the real sign-up
# needs a solved hCaptcha and sends an email) and to clean up after itself.
# Everything a run creates is deleted at the end.
#
# Captcha / email (see README "End-to-end tests"):
#   dev/prod - real captcha and real emails, so the sign-up tests are skipped;
#              on prod, captcha must be enforced (a bogus token must get 403).
#   local    - pass CAPTCHA_TOKEN when the local API runs with hCaptcha's test
#              keys to exercise the real sign-up. If .env.local sets
#              MAILTRAP_ACCOUNT_ID + MAILTRAP_INBOX_ID (and the API sends to that
#              sandbox inbox), magic links are read from the inbox (token:
#              MAILTRAP_INBOX_API_TOKEN, else EMAIL_SENDER_TOKEN_MAILTRAP).
set -euo pipefail

ENV_NAME="${1:-}"
shift || true

case "$ENV_NAME" in
  local) DEFAULT_API_URL="http://localhost:8080";      DEFAULT_CORS_ORIGIN="http://localhost:8081"; DEFAULT_METRICS_URL="http://localhost:9091" ;;
  dev)   DEFAULT_API_URL="https://api-dev.rottenbik.es"; DEFAULT_CORS_ORIGIN="https://dev.rottenbik.es"; DEFAULT_METRICS_URL="" ;;
  prod)  DEFAULT_API_URL="https://api.rottenbik.es";     DEFAULT_CORS_ORIGIN="https://rottenbik.es";     DEFAULT_METRICS_URL="" ;;
  *) echo "Usage: $0 <local|dev|prod> [go test args...]"; exit 1 ;;
esac

# Don't let variables exported by the Makefile (from .env.local) leak into
# another environment's settings.
unset DATABASE_URL DB_USER DB_PASSWORD DB_HOST DB_PORT DB_NAME DB_SSLMODE \
  MAILTRAP_ACCOUNT_ID MAILTRAP_INBOX_ID MAILTRAP_INBOX_API_TOKEN EMAIL_SENDER_TOKEN_MAILTRAP

ENV_FILE=".env.$ENV_NAME"
if [ -f "$ENV_FILE" ]; then
  set -o allexport
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +o allexport
elif [ "$ENV_NAME" != "local" ]; then
  echo "Error: $ENV_FILE not found (needed for the $ENV_NAME database settings)."
  exit 1
fi

if [ -z "${DATABASE_URL:-}" ]; then
  DB_PASSWORD_ENCODED=$(python3 -c "import urllib.parse, sys; print(urllib.parse.quote(sys.argv[1], safe=''))" "${DB_PASSWORD:-rottenbikes}")
  DATABASE_URL="postgres://${DB_USER:-rottenbikes}:${DB_PASSWORD_ENCODED}@${DB_HOST:-localhost}:${DB_PORT:-5432}/${DB_NAME:-rottenbikes}?sslmode=${DB_SSLMODE:-disable}"
fi

export E2E_ENV="$ENV_NAME"
export E2E_API_URL="${API_URL:-$DEFAULT_API_URL}"
export E2E_DB_DSN="$DATABASE_URL"
export E2E_CORS_ORIGIN="${CORS_ORIGIN:-$DEFAULT_CORS_ORIGIN}"
export E2E_METRICS_URL="${METRICS_URL:-$DEFAULT_METRICS_URL}"
export E2E_CAPTCHA_TOKEN="${CAPTCHA_TOKEN:-}"
# Test accounts (hidden, reserved 6-digit bike numbers): off on local, on for
# dev/prod unless TEST_ACCOUNTS=0/1 says otherwise.
if [ -n "${TEST_ACCOUNTS:-}" ]; then
  export E2E_TEST_ACCOUNTS="$TEST_ACCOUNTS"
fi
if [ "$ENV_NAME" = "prod" ]; then
  export E2E_REQUIRE_CAPTCHA=1
fi
if [ -n "${MAILTRAP_ACCOUNT_ID:-}" ] && [ -n "${MAILTRAP_INBOX_ID:-}" ]; then
  export E2E_MAILTRAP_ACCOUNT_ID="$MAILTRAP_ACCOUNT_ID"
  export E2E_MAILTRAP_INBOX_ID="$MAILTRAP_INBOX_ID"
  export E2E_MAILTRAP_API_TOKEN="${MAILTRAP_INBOX_API_TOKEN:-${EMAIL_SENDER_TOKEN_MAILTRAP:-}}"
fi

if [ "$ENV_NAME" = "prod" ] && [ "${CONFIRM:-}" != "prod" ]; then
  echo "Refusing to run against prod without CONFIRM=prod."
  echo "The suite creates (and then deletes) test users, bikes and reviews, and temporarily promotes test users to admin."
  exit 1
fi

echo "Running E2E suite against $ENV_NAME ($E2E_API_URL)..."
exec go test -tags e2e -count=1 -v ./e2e/... "$@"
