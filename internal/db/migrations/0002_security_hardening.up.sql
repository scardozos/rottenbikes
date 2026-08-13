-- Security hardening phase 1.
--
-- 1) Decouple the magic-link confirm credential from the poll credential.
--    Previously the request endpoint returned HashToken(magicToken) to the
--    requester, and the poll endpoint matched that same hash against
--    magic_links.token (the confirm key). Whoever requested a link could
--    therefore poll until the victim clicked the emailed link and harvest the
--    victim's API token (cross-device account takeover).
--    We now store a separate, per-request `poll_token` (hashed). The request
--    endpoint returns the RAW poll token to the requesting device; only that
--    device can poll. The emailed (magic) token cannot poll, and the poll token
--    cannot confirm.
ALTER TABLE magic_links ADD COLUMN poll_token TEXT;

-- 2) Hash API tokens at rest. The long-lived bearer token in posters.api_token
--    is now stored as a SHA-256 hex hash. A database read leak no longer
--    discloses usable session tokens. The raw token is only ever held
--    transiently in magic_links.api_token (gated by poll_token + expiry) for
--    one-time poll retrieval and returned to the client at confirm time.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Backfill: hash any existing plaintext api tokens in place. Storage held the
-- raw token, so sha256(raw) is exactly what the app now expects to compare
-- against. Idempotent under golang-migrate's once-only semantics.
UPDATE posters
SET api_token = encode(digest(api_token, 'sha256'), 'hex')
WHERE api_token IS NOT NULL;

-- Enforce uniqueness and make the per-request bearer lookup an index scan
-- instead of a sequential scan (every authenticated request hits this).
CREATE UNIQUE INDEX idx_posters_api_token ON posters (api_token) WHERE api_token IS NOT NULL;

-- Enforce uniqueness and make the per-request poll lookup an index scan.
CREATE UNIQUE INDEX idx_magic_links_poll_token ON magic_links (poll_token) WHERE poll_token IS NOT NULL;