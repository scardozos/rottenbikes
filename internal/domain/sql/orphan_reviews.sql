UPDATE reviews
SET poster_id = NULL
WHERE poster_id = $1
