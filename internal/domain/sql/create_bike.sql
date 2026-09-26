INSERT INTO bikes (numerical_id, hash_id, is_electric, was_scanned, creator_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING numerical_id, hash_id, is_electric, was_scanned, created_ts, updated_ts
