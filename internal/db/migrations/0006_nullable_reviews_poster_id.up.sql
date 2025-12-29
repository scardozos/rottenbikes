ALTER TABLE reviews DROP CONSTRAINT fk_reviews_poster;
ALTER TABLE reviews ALTER COLUMN poster_id DROP NOT NULL;
ALTER TABLE reviews ADD CONSTRAINT fk_reviews_poster FOREIGN KEY (poster_id) REFERENCES posters (poster_id) ON DELETE SET NULL;
