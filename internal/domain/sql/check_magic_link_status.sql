SELECT api_token
FROM magic_links
WHERE poll_token = $1
  AND consumed_ts IS NOT NULL
  AND expires_ts > NOW()