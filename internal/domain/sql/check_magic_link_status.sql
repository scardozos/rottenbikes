SELECT api_token
FROM magic_links
WHERE token = $1 AND consumed_ts IS NOT NULL
