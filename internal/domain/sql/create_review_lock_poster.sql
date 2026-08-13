SELECT poster_id
FROM posters
WHERE poster_id = $1
FOR UPDATE