SELECT created_ts
FROM reviews
WHERE poster_id = $1 AND bike_numerical_id = $2
ORDER BY created_ts DESC
LIMIT 1
