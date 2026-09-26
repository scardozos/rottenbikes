ALTER TABLE posters ADD COLUMN IF NOT EXISTS api_token TEXT;
ALTER TABLE posters ADD COLUMN IF NOT EXISTS api_token_expires_ts TIMESTAMPTZ;

-- Restore most recent token if available
UPDATE posters p
SET api_token = pt.token_hash,
    api_token_expires_ts = pt.expires_ts
FROM (
    SELECT DISTINCT ON (poster_id) poster_id, token_hash, expires_ts
    FROM poster_tokens
    ORDER BY poster_id, created_ts DESC
) pt
WHERE p.poster_id = pt.poster_id;

DROP TABLE IF EXISTS poster_tokens;
