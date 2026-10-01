-- Login codes replace cross-device polling.
--
-- Polling let whoever requested a link receive the session once the email's
-- recipient clicked it, so an attacker could request a link for a victim's
-- (public) username and wait. Now the emailed link logs in only the device
-- that opens it, and another device logs in by entering the 6-digit code from
-- the same email together with its request token (magic_links.poll_token).
--
-- code_hash is SHA-256 of the code. Redeeming it also needs the raw request
-- token, which is only stored hashed, so a leaked code_hash is not usable.
-- The link (consumed_ts) and the code (code_used_ts) are each usable once,
-- independently: tapping the link on a phone must not stop the code from
-- logging in the laptop that asked for the email.
ALTER TABLE magic_links ADD COLUMN code_hash TEXT;
ALTER TABLE magic_links ADD COLUMN code_attempts INT NOT NULL DEFAULT 0;
ALTER TABLE magic_links ADD COLUMN code_used_ts TIMESTAMPTZ;

-- Raw API tokens are no longer parked here for polling. Clear the ones left by
-- links that were confirmed but never polled. The column itself is dropped in
-- a later migration, once no running API version writes to it.
UPDATE magic_links SET api_token = NULL WHERE api_token IS NOT NULL;
