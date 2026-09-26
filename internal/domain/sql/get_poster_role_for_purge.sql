-- Lock the target's row so a concurrent promote/demote serializes with the
-- purge (no TOCTOU): the purge either sees the committed role or waits.
SELECT role
FROM posters
WHERE poster_id = $1
FOR UPDATE
