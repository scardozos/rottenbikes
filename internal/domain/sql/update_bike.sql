-- hash_id: NULL keeps the current value, '' clears it (like on create).
UPDATE bikes
SET
	hash_id     = CASE WHEN $1::text IS NULL THEN hash_id ELSE NULLIF($1::text, '') END,
	is_electric = COALESCE($2, is_electric),
	updated_ts  = NOW()
WHERE numerical_id = $3 AND creator_id = $4
