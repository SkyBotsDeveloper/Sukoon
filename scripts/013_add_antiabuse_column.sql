-- Add antiabuse_enabled column to chat_settings
ALTER TABLE chat_settings 
ADD COLUMN IF NOT EXISTS antiabuse_enabled BOOLEAN DEFAULT FALSE;

-- Create index for faster lookups
CREATE INDEX IF NOT EXISTS idx_chat_settings_antiabuse 
ON chat_settings(chat_id) 
WHERE antiabuse_enabled = TRUE;
