ALTER TABLE magic_links DROP COLUMN IF EXISTS code_used_ts;
ALTER TABLE magic_links DROP COLUMN IF EXISTS code_attempts;
ALTER TABLE magic_links DROP COLUMN IF EXISTS code_hash;
