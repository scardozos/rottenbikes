SELECT api_token, email, api_token_expires_ts
FROM posters
WHERE poster_id = $1
FOR UPDATE
