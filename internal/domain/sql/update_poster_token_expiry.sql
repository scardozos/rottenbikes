UPDATE posters
SET api_token_expires_ts = $1
WHERE poster_id = $2
