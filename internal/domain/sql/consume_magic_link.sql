UPDATE magic_links
SET consumed_ts = NOW(), api_token = $1
WHERE token = $2
