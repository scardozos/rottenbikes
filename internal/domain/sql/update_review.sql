UPDATE reviews
SET comment = COALESCE($1, comment),
	bike_img = COALESCE($2, bike_img)
WHERE review_id = $3
