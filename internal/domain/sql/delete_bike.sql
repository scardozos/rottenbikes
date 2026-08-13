DELETE FROM bikes
WHERE numerical_id = $1 AND creator_id = $2
