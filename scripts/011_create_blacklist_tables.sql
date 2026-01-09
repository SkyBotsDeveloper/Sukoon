-- Create blacklisted_chats table
CREATE TABLE IF NOT EXISTS blacklisted_chats (
  chat_id BIGINT PRIMARY KEY,
  reason TEXT,
  blacklisted_by BIGINT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create blacklisted_users table  
CREATE TABLE IF NOT EXISTS blacklisted_users (
  user_id BIGINT PRIMARY KEY,
  reason TEXT,
  blacklisted_by BIGINT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_blacklisted_chats_created ON blacklisted_chats(created_at);
CREATE INDEX IF NOT EXISTS idx_blacklisted_users_created ON blacklisted_users(created_at);
