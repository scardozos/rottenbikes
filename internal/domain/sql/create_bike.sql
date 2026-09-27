-- is_test is inherited from the creator (see migration 0010).
INSERT INTO bikes (numerical_id, hash_id, is_electric, was_scanned, creator_id, is_test)
VALUES ($1, $2, $3, $4, $5, COALESCE((SELECT is_test FROM posters WHERE poster_id = $5), FALSE))
RETURNING numerical_id, hash_id, is_electric, was_scanned, created_ts, updated_ts
