-- Add language column to chat_settings if not exists
ALTER TABLE chat_settings ADD COLUMN IF NOT EXISTS language VARCHAR(10) DEFAULT 'en';

-- Create index for faster language lookups
CREATE INDEX IF NOT EXISTS idx_chat_settings_language ON chat_settings(language);

-- Update existing rows to have default language
UPDATE chat_settings SET language = 'en' WHERE language IS NULL;
