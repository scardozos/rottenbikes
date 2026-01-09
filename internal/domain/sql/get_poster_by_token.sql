SELECT poster_id, email, username, api_token_expires_ts, email_verified
FROM posters
WHERE api_token = $1
