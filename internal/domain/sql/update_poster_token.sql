UPDATE posters
SET api_token = $1, api_token_expires_ts = $2
WHERE poster_id = $3
