SELECT poster_id, email, username, role, created_ts
FROM posters
WHERE role = 'admin'
ORDER BY created_ts
