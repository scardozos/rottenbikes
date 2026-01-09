INSERT INTO review_ratings (review_id, subcategory, score)
VALUES ($1, $2, $3)
ON CONFLICT (review_id, subcategory)
DO UPDATE SET score = EXCLUDED.score
