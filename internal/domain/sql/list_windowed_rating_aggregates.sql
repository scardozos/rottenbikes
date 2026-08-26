SELECT
	rr.subcategory,
	ROUND(AVG(CASE WHEN r.created_ts >= NOW() - INTERVAL '1 week' THEN rr.score END)::numeric, 2) as avg_1w,
	ROUND(AVG(CASE WHEN r.created_ts >= NOW() - INTERVAL '2 weeks' THEN rr.score END)::numeric, 2) as avg_2w,
	ROUND(AVG(rr.score)::numeric, 2) as avg_overall
FROM review_ratings rr
JOIN reviews r ON rr.review_id = r.review_id
WHERE r.bike_numerical_id = $1
GROUP BY rr.subcategory
ORDER BY rr.subcategory
