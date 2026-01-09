-- Global Bans table
CREATE TABLE IF NOT EXISTS global_bans (
  user_id BIGINT PRIMARY KEY,
  reason TEXT,
  banned_by BIGINT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Sudo Users table
CREATE TABLE IF NOT EXISTS sudo_users (
  user_id BIGINT PRIMARY KEY,
  added_by BIGINT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- AFK Users table
CREATE TABLE IF NOT EXISTS afk_users (
  user_id BIGINT PRIMARY KEY,
  reason TEXT,
  afk_time TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_global_bans_user ON global_bans(user_id);
CREATE INDEX IF NOT EXISTS idx_sudo_users_user ON sudo_users(user_id);
CREATE INDEX IF NOT EXISTS idx_afk_users_user ON afk_users(user_id);
