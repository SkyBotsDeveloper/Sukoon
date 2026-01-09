-- Table for storing cloned bot instances
CREATE TABLE IF NOT EXISTS bot_clones (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    bot_token TEXT NOT NULL UNIQUE,
    bot_username TEXT,
    bot_name TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for token lookup in webhook
CREATE INDEX IF NOT EXISTS idx_bot_clones_token ON bot_clones(bot_token);
