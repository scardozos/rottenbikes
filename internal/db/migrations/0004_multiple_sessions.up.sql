-- Multi-session support: decouple API tokens from posters table
CREATE TABLE poster_tokens (
    token_hash TEXT PRIMARY KEY,
    poster_id BIGINT NOT NULL REFERENCES posters(poster_id) ON DELETE CASCADE,
    created_ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_ts TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_poster_tokens_poster ON poster_tokens(poster_id);
CREATE INDEX idx_poster_tokens_expires ON poster_tokens(expires_ts);

-- Backfill existing valid hashed tokens
INSERT INTO poster_tokens (token_hash, poster_id, expires_ts)
SELECT api_token, poster_id, api_token_expires_ts
FROM posters
WHERE api_token IS NOT NULL AND (api_token_expires_ts IS NULL OR api_token_expires_ts > NOW());

-- Remove token columns from posters table
ALTER TABLE posters DROP COLUMN IF EXISTS api_token;
ALTER TABLE posters DROP COLUMN IF EXISTS api_token_expires_ts;
