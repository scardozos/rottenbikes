UPDATE bikes
SET creator_id = NULL
WHERE creator_id = $1
