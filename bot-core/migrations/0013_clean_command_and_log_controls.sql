ALTER TABLE chat_settings
    ADD COLUMN IF NOT EXISTS clean_command_all BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS clean_command_admin BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS clean_command_user BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS clean_command_other BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS log_category_settings BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS log_category_admin BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS log_category_user BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS log_category_automated BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS log_category_reports BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS log_category_other BOOLEAN NOT NULL DEFAULT TRUE;

UPDATE chat_settings
SET clean_command_all = TRUE
WHERE clean_commands = TRUE
  AND clean_command_all = FALSE
  AND clean_command_admin = FALSE
  AND clean_command_user = FALSE
  AND clean_command_other = FALSE;

UPDATE chat_settings
SET clean_commands = clean_command_all OR clean_command_admin OR clean_command_user OR clean_command_other;
