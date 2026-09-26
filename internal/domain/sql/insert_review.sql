-- was_scanned is computed server-side from scan_events, so clients cannot spoof it.
INSERT INTO reviews (poster_id, bike_numerical_id, bike_img, comment, was_scanned)
VALUES ($1, $2, $3, $4, EXISTS (
	SELECT 1 FROM scan_events
	WHERE poster_id = $1 AND bike_numerical_id = $2
))
RETURNING review_id
