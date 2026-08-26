SELECT 
	b.numerical_id, 
	b.hash_id, 
	b.is_electric, 
	b.created_ts, 
	b.updated_ts,
	ra.average_rating
FROM bikes b
LEFT JOIN rating_aggregates ra 
	ON b.numerical_id = ra.bike_numerical_id 
	AND ra.subcategory = 'overall'
LEFT JOIN LATERAL (
    SELECT COUNT(*) as review_count, MAX(created_ts) as max_created_ts 
    FROM reviews r 
    WHERE r.bike_numerical_id = b.numerical_id
) rs ON $4::text IN ('most_reviewed', 'recent')
WHERE 
	($3::text = '' OR b.numerical_id ILIKE '%' || $3::text || '%' OR b.hash_id ILIKE '%' || $3::text || '%')
ORDER BY 
    CASE WHEN $4::text = 'rating' THEN ra.average_rating END DESC NULLS LAST,
    CASE WHEN $4::text = 'most_reviewed' THEN COALESCE(rs.review_count, 0) END DESC,
    CASE WHEN $4::text = 'recent' THEN COALESCE(rs.max_created_ts, b.created_ts) END DESC,
	b.numerical_id
LIMIT (CASE WHEN $1::int >= 0 THEN $1::int ELSE NULL END)
OFFSET (CASE WHEN $2::int >= 0 THEN $2::int ELSE 0 END)
