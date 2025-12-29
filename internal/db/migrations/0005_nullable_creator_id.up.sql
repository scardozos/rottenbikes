ALTER TABLE bikes DROP CONSTRAINT fk_bikes_creator;
ALTER TABLE bikes ALTER COLUMN creator_id DROP NOT NULL;
ALTER TABLE bikes ADD CONSTRAINT fk_bikes_creator FOREIGN KEY (creator_id) REFERENCES posters (poster_id) ON DELETE SET NULL;
