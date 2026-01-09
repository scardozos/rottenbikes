SELECT COUNT(*) FROM magic_links
WHERE poster_id = $1 AND created_ts > NOW() - INTERVAL '24 hours'
