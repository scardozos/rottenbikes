ALTER TABLE moderation_actions DROP CONSTRAINT IF EXISTS moderation_actions_one_target;
DELETE FROM moderation_actions WHERE target_poster_id IS NULL;
ALTER TABLE moderation_actions ALTER COLUMN target_poster_id SET NOT NULL;
ALTER TABLE moderation_actions DROP COLUMN IF EXISTS target_bike_id;
