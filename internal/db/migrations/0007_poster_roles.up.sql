-- Admin roles: promotion/demotion is managed out-of-band (cmd/adminctl),
-- so admin identity lives only in the database — never in git/env/configmaps.
-- The role is read on every token lookup, so demotion takes effect instantly.
CREATE TYPE poster_role AS ENUM ('user', 'admin');

ALTER TABLE posters ADD COLUMN role poster_role NOT NULL DEFAULT 'user';
