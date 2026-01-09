SELECT bike_numerical_id, subcategory, average_rating
FROM rating_aggregates
WHERE bike_numerical_id = $1
ORDER BY subcategory
