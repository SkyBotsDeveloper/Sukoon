ALTER TABLE chat_settings
    ADD COLUMN IF NOT EXISTS blocklist_action TEXT NOT NULL DEFAULT 'nothing',
    ADD COLUMN IF NOT EXISTS blocklist_action_duration_seconds INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS blocklist_delete BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS blocklist_reason TEXT NOT NULL DEFAULT '';

ALTER TABLE blocklist_rules
    ADD COLUMN IF NOT EXISTS action_duration_seconds INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS delete_behavior TEXT NOT NULL DEFAULT 'inherit',
    ADD COLUMN IF NOT EXISTS reason TEXT NOT NULL DEFAULT '';

UPDATE blocklist_rules
SET action = ''
WHERE action = 'delete';
