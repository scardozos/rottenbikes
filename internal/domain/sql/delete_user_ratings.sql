DELETE FROM review_ratings
WHERE review_id IN (SELECT review_id FROM reviews WHERE poster_id = $1)
