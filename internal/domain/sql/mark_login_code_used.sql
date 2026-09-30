UPDATE magic_links
SET code_used_ts = NOW()
WHERE id = $1
