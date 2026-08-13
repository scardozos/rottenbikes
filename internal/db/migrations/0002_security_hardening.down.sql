-- Reverse the schema changes from 0002_security_hardening.up.sql.
-- NOTE: the api_token one-way hash backfill cannot be reversed (sha256 is not
-- invertible). After running this down migration the stored api_token values
-- remain hashed; clients that still hold the raw token continue to work because
-- the app hashes the incoming bearer before comparing.
DROP INDEX IF EXISTS idx_magic_links_poll_token;
DROP INDEX IF EXISTS idx_posters_api_token;
ALTER TABLE magic_links DROP COLUMN IF EXISTS poll_token;
DROP EXTENSION IF EXISTS pgcrypto;