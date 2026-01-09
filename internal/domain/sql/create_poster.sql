INSERT INTO posters (email, username)
VALUES ($1, $2)
RETURNING poster_id, api_token, api_token_expires_ts
