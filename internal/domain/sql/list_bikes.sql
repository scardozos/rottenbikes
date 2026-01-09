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
ORDER BY b.numerical_id
