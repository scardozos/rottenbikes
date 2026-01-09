SELECT poster_id, expires_ts, consumed_ts
FROM magic_links
WHERE token = $1
FOR UPDATE
