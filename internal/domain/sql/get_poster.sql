SELECT poster_id, api_token, api_token_expires_ts, email
FROM posters
WHERE email = $1 OR username = $1
FOR UPDATE
