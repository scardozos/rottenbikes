ALTER TABLE bikes DROP CONSTRAINT fk_bikes_creator;
DELETE FROM bikes WHERE creator_id IS NULL;
ALTER TABLE bikes ALTER COLUMN creator_id SET NOT NULL;
ALTER TABLE bikes ADD CONSTRAINT fk_bikes_creator FOREIGN KEY (creator_id) REFERENCES posters (poster_id) ON DELETE RESTRICT;
