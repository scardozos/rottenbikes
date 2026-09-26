WITH updated_poster AS (
    UPDATE posters
    SET email_verified = TRUE
    WHERE poster_id = $3
    RETURNING email
), inserted_token AS (
    INSERT INTO poster_tokens (token_hash, poster_id, expires_ts)
    VALUES ($1, $3, $2)
    RETURNING expires_ts
)
SELECT inserted_token.expires_ts, updated_poster.email
FROM inserted_token, updated_poster