INSERT INTO rating_aggregates (
	bike_numerical_id, subcategory, rating_sum, rating_count, average_rating
)
SELECT
	r.bike_numerical_id,
	rr.subcategory,
	SUM(rr.score)                        AS rating_sum,
	COUNT(*)                             AS rating_count,
	ROUND(AVG(rr.score)::numeric, 2)     AS average_rating
FROM review_ratings rr
JOIN reviews r ON rr.review_id = r.review_id
WHERE r.bike_numerical_id = $1
GROUP BY r.bike_numerical_id, rr.subcategory
