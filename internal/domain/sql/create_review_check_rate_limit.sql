SELECT COUNT(*)
FROM reviews
WHERE poster_id = $1 AND created_ts > NOW() - INTERVAL '1 hour'
