ALTER TABLE antiflood_settings
    ADD COLUMN IF NOT EXISTS timed_limit INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS action_duration_seconds INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS clear_all BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE antiflood_settings
    ALTER COLUMN flood_limit SET DEFAULT 0;

UPDATE antiflood_settings
SET flood_limit = 0
WHERE enabled = FALSE;
