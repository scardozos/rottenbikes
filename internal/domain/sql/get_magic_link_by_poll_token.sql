SELECT id, poster_id, code_hash, code_attempts, expires_ts, code_used_ts
FROM magic_links
WHERE poll_token = $1
FOR UPDATE
