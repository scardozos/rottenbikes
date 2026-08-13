WITH paginated_reviews AS (
	SELECT review_id, poster_id, bike_numerical_id, comment, created_ts, bike_img
	FROM reviews
	WHERE poster_id = $1
	ORDER BY created_ts DESC
	LIMIT (CASE WHEN $2 >= 0 THEN $2 ELSE NULL END)
	OFFSET (CASE WHEN $3 >= 0 THEN $3 ELSE 0 END)
)
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
FROM paginated_reviews r
LEFT JOIN posters p         ON p.poster_id = r.poster_id
LEFT JOIN review_ratings rr ON rr.review_id = r.review_id
ORDER BY r.created_ts DESC, r.review_id DESC, rr.subcategory
