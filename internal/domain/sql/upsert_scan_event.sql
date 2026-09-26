INSERT INTO scan_events (poster_id, bike_numerical_id)
VALUES ($1, $2)
ON CONFLICT (poster_id, bike_numerical_id)
DO UPDATE SET scanned_ts = NOW()
