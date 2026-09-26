-- Audit log for moderation actions performed via the admin API.
-- No FKs on purpose: audit rows must survive the purge of their target.
CREATE TABLE moderation_actions (
    id               BIGSERIAL PRIMARY KEY,
    admin_poster_id  BIGINT      NOT NULL,
    action           TEXT        NOT NULL,
    target_poster_id BIGINT      NOT NULL,
    created_ts       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_moderation_actions_created ON moderation_actions (created_ts);
