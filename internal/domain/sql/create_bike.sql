INSERT INTO bikes (numerical_id, hash_id, is_electric, creator_id)
VALUES ($1, $2, $3, $4)
RETURNING numerical_id, hash_id, is_electric, created_ts, updated_ts
