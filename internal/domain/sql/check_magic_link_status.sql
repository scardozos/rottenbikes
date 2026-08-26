WITH target AS (
    SELECT id, api_token
    FROM magic_links
    WHERE poll_token = $1
      AND consumed_ts IS NOT NULL
      AND expires_ts > NOW()
      AND api_token IS NOT NULL
    FOR UPDATE
)
UPDATE magic_links
SET api_token = NULL
FROM target
WHERE magic_links.id = target.id
RETURNING target.api_token;