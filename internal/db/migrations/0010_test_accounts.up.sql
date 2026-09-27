-- Accounts used by the E2E suite against shared environments. Flagged only
-- directly in the database (never through the API). Bikes inherit the flag
-- from their creator at insert time and keep it even if the creator is later
-- deleted, so test bikes stay out of public listings.
ALTER TABLE posters ADD COLUMN is_test BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE bikes ADD COLUMN is_test BOOLEAN NOT NULL DEFAULT FALSE;
