-- Add CAPTCHA settings table
CREATE TABLE IF NOT EXISTS captcha_settings (
  id SERIAL PRIMARY KEY,
  chat_id BIGINT UNIQUE NOT NULL,
  captcha_enabled BOOLEAN DEFAULT false,
  captcha_mode VARCHAR(20) DEFAULT 'button',
  mute_time INTEGER DEFAULT 300,
  kick_on_fail BOOLEAN DEFAULT false,
  button_text VARCHAR(100) DEFAULT 'Click to verify',
  require_rules BOOLEAN DEFAULT false,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create index
CREATE INDEX IF NOT EXISTS idx_captcha_settings_chat_id ON captcha_settings(chat_id);

-- Add to existing tables if not exists
DO $$
BEGIN
  -- Ensure flood_tracking table exists
  CREATE TABLE IF NOT EXISTS flood_tracking (
    id SERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    message_count INTEGER DEFAULT 1,
    last_message_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(chat_id, user_id)
  );
EXCEPTION
  WHEN duplicate_table THEN NULL;
END $$;

-- Ensure disabled_commands table exists
CREATE TABLE IF NOT EXISTS disabled_commands (
  id SERIAL PRIMARY KEY,
  chat_id BIGINT NOT NULL,
  command VARCHAR(50) NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  UNIQUE(chat_id, command)
);

-- Ensure report_settings table exists
CREATE TABLE IF NOT EXISTS report_settings (
  id SERIAL PRIMARY KEY,
  chat_id BIGINT UNIQUE NOT NULL,
  reports_enabled BOOLEAN DEFAULT true,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Ensure log_channels table exists
CREATE TABLE IF NOT EXISTS log_channels (
  id SERIAL PRIMARY KEY,
  chat_id BIGINT UNIQUE NOT NULL,
  log_channel_id BIGINT NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Ensure connections table exists
CREATE TABLE IF NOT EXISTS connections (
  id SERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  chat_id BIGINT NOT NULL,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  UNIQUE(user_id, chat_id)
);

-- Ensure approved_users table exists
CREATE TABLE IF NOT EXISTS approved_users (
  id SERIAL PRIMARY KEY,
  chat_id BIGINT NOT NULL,
  user_id BIGINT NOT NULL,
  approved_by BIGINT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  UNIQUE(chat_id, user_id)
);

-- Ensure global_bans table exists
CREATE TABLE IF NOT EXISTS global_bans (
  id SERIAL PRIMARY KEY,
  user_id BIGINT UNIQUE NOT NULL,
  banned_by BIGINT,
  reason TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Ensure sudoers table exists
CREATE TABLE IF NOT EXISTS sudoers (
  id SERIAL PRIMARY KEY,
  user_id BIGINT UNIQUE NOT NULL,
  added_by BIGINT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
