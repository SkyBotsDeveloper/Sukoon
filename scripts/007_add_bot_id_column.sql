-- Add bot_id column to bot_clones if it doesn't exist
ALTER TABLE bot_clones ADD COLUMN IF NOT EXISTS bot_id BIGINT;

-- Create index for faster lookups
CREATE INDEX IF NOT EXISTS idx_bot_clones_bot_id ON bot_clones(bot_id);
CREATE INDEX IF NOT EXISTS idx_bot_clones_user_id ON bot_clones(user_id);
