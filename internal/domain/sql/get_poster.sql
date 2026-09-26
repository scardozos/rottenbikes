SELECT poster_id, email
FROM posters
WHERE email = $1 OR username = $1
FOR UPDATE
