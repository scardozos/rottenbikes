-- Audit admin bike deletions: a moderation action targets either a poster or
-- a bike. No FK on purpose, like target_poster_id: audit rows must survive
-- the deletion of their target.
ALTER TABLE moderation_actions ADD COLUMN target_bike_id TEXT;
ALTER TABLE moderation_actions ALTER COLUMN target_poster_id DROP NOT NULL;
ALTER TABLE moderation_actions ADD CONSTRAINT moderation_actions_one_target
    CHECK ((target_poster_id IS NULL) <> (target_bike_id IS NULL));
