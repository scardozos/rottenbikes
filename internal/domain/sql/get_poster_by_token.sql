SELECT p.poster_id, p.email, p.username, p.role, pt.expires_ts AS api_token_expires_ts, p.email_verified, p.is_test
FROM poster_tokens pt
JOIN posters p ON p.poster_id = pt.poster_id
WHERE pt.token_hash = $1
