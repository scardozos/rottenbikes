-- An empty hash_id means "no QR code" and must be stored as NULL: bikes.hash_id
-- is UNIQUE, so a second '' would collide. PUT /bikes/{id} used to store ''
-- when clearing the hash; normalize those rows and forbid '' from now on.
UPDATE bikes SET hash_id = NULL WHERE hash_id = '';

ALTER TABLE bikes ADD CONSTRAINT bikes_hash_id_not_empty CHECK (hash_id <> '');
