ALTER TABLE reviews DROP CONSTRAINT fk_reviews_poster;
DELETE FROM reviews WHERE poster_id IS NULL;
ALTER TABLE reviews ALTER COLUMN poster_id SET NOT NULL;
ALTER TABLE reviews ADD CONSTRAINT fk_reviews_poster FOREIGN KEY (poster_id) REFERENCES posters (poster_id) ON DELETE CASCADE;
