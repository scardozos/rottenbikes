UPDATE posters
SET role = $1::poster_role
WHERE email = $2 OR username = $2
RETURNING poster_id, email, username, role
