SELECT
	p.poster_id,
	p.username,
	p.email,
	p.created_ts,
	(SELECT COUNT(*) FROM reviews r WHERE r.poster_id = p.poster_id) AS review_count,
	(SELECT COUNT(*) FROM bikes b WHERE b.creator_id = p.poster_id) AS bike_count
FROM posters p
WHERE p.email ILIKE '%' || $1 || '%' OR p.username ILIKE '%' || $1 || '%'
ORDER BY p.created_ts DESC
LIMIT (CASE WHEN $2::int >= 0 THEN $2::int ELSE 50 END)
