INSERT INTO reviews (poster_id, bike_numerical_id, bike_img, comment)
VALUES ($1, $2, $3, $4)
RETURNING review_id
