UPDATE magic_links
SET code_attempts = code_attempts + 1
WHERE id = $1
