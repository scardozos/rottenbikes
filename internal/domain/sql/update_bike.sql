UPDATE bikes
SET
	hash_id     = COALESCE($1, hash_id),
	is_electric = COALESCE($2, is_electric),
	updated_ts  = NOW()
WHERE numerical_id = $3 AND creator_id = $4
