UPDATE posters
SET api_token = $1, api_token_expires_ts = $2, email_verified = TRUE
WHERE poster_id = $3
RETURNING api_token_expires_ts, email