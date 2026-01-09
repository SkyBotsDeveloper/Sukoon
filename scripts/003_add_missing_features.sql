-- ============================================
-- ANTIRAID TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS antiraid_settings (
    chat_id BIGINT PRIMARY KEY REFERENCES chats(chat_id) ON DELETE CASCADE,
    antiraid_enabled BOOLEAN DEFAULT FALSE,
    raid_duration INTEGER DEFAULT 21600, -- 6 hours in seconds
    raid_action_time INTEGER DEFAULT 3600, -- 1 hour in seconds
    auto_raid_threshold INTEGER DEFAULT 0, -- joins per minute (0 = disabled)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- CLEAN COMMANDS TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS clean_commands_settings (
    chat_id BIGINT PRIMARY KEY REFERENCES chats(chat_id) ON DELETE CASCADE,
    clean_all BOOLEAN DEFAULT FALSE, -- delete all commands
    clean_admin BOOLEAN DEFAULT FALSE, -- delete admin commands
    clean_user BOOLEAN DEFAULT FALSE, -- delete user commands
    clean_other BOOLEAN DEFAULT FALSE, -- delete other bot commands
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- CLEAN SERVICE TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS clean_service_settings (
    chat_id BIGINT PRIMARY KEY REFERENCES chats(chat_id) ON DELETE CASCADE,
    clean_service_enabled BOOLEAN DEFAULT FALSE,
    clean_join BOOLEAN DEFAULT FALSE, -- "X joined" messages
    clean_leave BOOLEAN DEFAULT FALSE, -- "X left" messages
    clean_pin BOOLEAN DEFAULT FALSE, -- "X pinned a message"
    clean_title BOOLEAN DEFAULT FALSE, -- title change messages
    clean_photo BOOLEAN DEFAULT FALSE, -- photo change messages
    clean_video_chat BOOLEAN DEFAULT FALSE, -- video chat messages
    clean_other BOOLEAN DEFAULT FALSE, -- other service messages
    clean_all BOOLEAN DEFAULT FALSE, -- clean all service messages
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- DISABLE SETTINGS TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS disable_settings (
    chat_id BIGINT PRIMARY KEY REFERENCES chats(chat_id) ON DELETE CASCADE,
    disable_admin BOOLEAN DEFAULT FALSE, -- stop admins from using disabled commands
    disable_delete BOOLEAN DEFAULT TRUE, -- delete disabled commands
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- UPDATE CAPTCHA TABLE
-- ============================================

ALTER TABLE captcha_settings ADD COLUMN IF NOT EXISTS captcha_mute_time INTEGER DEFAULT 0;
ALTER TABLE captcha_settings ADD COLUMN IF NOT EXISTS captcha_kick_time INTEGER DEFAULT 0;

-- ============================================
-- FEDERATION OWNER SETTINGS
-- ============================================

CREATE TABLE IF NOT EXISTS federation_settings (
    fed_id UUID PRIMARY KEY REFERENCES federations(fed_id) ON DELETE CASCADE,
    fed_notify BOOLEAN DEFAULT TRUE,
    fed_require_reason BOOLEAN DEFAULT FALSE,
    fed_quiet BOOLEAN DEFAULT FALSE,
    fed_log_channel BIGINT,
    fed_language TEXT DEFAULT 'en',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Federation subscriptions
CREATE TABLE IF NOT EXISTS federation_subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    fed_id UUID NOT NULL REFERENCES federations(fed_id) ON DELETE CASCADE,
    subscribed_to_fed_id UUID NOT NULL REFERENCES federations(fed_id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(fed_id, subscribed_to_fed_id)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_antiraid_settings_chat_id ON antiraid_settings(chat_id);
CREATE INDEX IF NOT EXISTS idx_clean_commands_settings_chat_id ON clean_commands_settings(chat_id);
CREATE INDEX IF NOT EXISTS idx_clean_service_settings_chat_id ON clean_service_settings(chat_id);
CREATE INDEX IF NOT EXISTS idx_disable_settings_chat_id ON disable_settings(chat_id);
CREATE INDEX IF NOT EXISTS idx_federation_subscriptions_fed_id ON federation_subscriptions(fed_id);
