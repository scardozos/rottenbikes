SELECT DISTINCT bike_numerical_id
FROM reviews
WHERE poster_id = $1
