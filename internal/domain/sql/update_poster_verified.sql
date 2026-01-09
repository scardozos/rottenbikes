UPDATE posters
SET email_verified = TRUE
WHERE poster_id = $1
RETURNING api_token_expires_ts, email
