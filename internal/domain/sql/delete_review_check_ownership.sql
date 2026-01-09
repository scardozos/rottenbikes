SELECT bike_numerical_id
FROM reviews
WHERE review_id = $1 AND poster_id = $2
