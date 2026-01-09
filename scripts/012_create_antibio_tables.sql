-- Create antibio_settings table
CREATE TABLE IF NOT EXISTS antibio_settings (
  chat_id TEXT PRIMARY KEY,
  enabled BOOLEAN DEFAULT false,
  updated_at TIMESTAMPTZ DEFAULT now()
);

-- Create antibio_free_users table (group-specific exemptions)
CREATE TABLE IF NOT EXISTS antibio_free_users (
  id SERIAL PRIMARY KEY,
  chat_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  added_by TEXT,
  created_at TIMESTAMPTZ DEFAULT now(),
  UNIQUE(chat_id, user_id)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_antibio_free_users_chat ON antibio_free_users(chat_id);
CREATE INDEX IF NOT EXISTS idx_antibio_free_users_user ON antibio_free_users(user_id);
CREATE INDEX IF NOT EXISTS idx_antibio_free_users_chat_user ON antibio_free_users(chat_id, user_id);
