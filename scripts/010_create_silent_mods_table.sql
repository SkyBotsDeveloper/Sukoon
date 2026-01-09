-- Create silent_mods table for silent power feature
CREATE TABLE IF NOT EXISTS silent_mods (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  chat_id BIGINT NOT NULL,
  user_id BIGINT NOT NULL,
  can_ban BOOLEAN DEFAULT false,
  can_mute BOOLEAN DEFAULT false,
  can_kick BOOLEAN DEFAULT false,
  can_warn BOOLEAN DEFAULT false,
  added_by BIGINT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(chat_id, user_id)
);

-- Create index for faster lookups
CREATE INDEX IF NOT EXISTS idx_silent_mods_chat_user ON silent_mods(chat_id, user_id);
