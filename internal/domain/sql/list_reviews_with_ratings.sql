SELECT
	r.review_id,
	r.poster_id,
	COALESCE(p.username, ''),
	r.bike_numerical_id,
	r.comment,
	r.created_ts,
	rr.subcategory,
	rr.score,
	r.bike_img
FROM reviews r
LEFT JOIN posters p       ON p.poster_id = r.poster_id
JOIN review_ratings rr ON rr.review_id = r.review_id
WHERE r.bike_numerical_id = $1
ORDER BY r.review_id, rr.subcategory
