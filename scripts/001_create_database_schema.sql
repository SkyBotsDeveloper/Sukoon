-- Sukoon Telegram Bot Database Schema
-- Complete database structure for all bot features

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================
-- CORE TABLES
-- ============================================

-- Chats table (groups/supergroups)
CREATE TABLE IF NOT EXISTS chats (
    chat_id BIGINT PRIMARY KEY,
    chat_type TEXT NOT NULL DEFAULT 'supergroup',
    chat_name TEXT,
    chat_username TEXT,
    language TEXT DEFAULT 'en',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Users table
CREATE TABLE IF NOT EXISTS users (
    user_id BIGINT PRIMARY KEY,
    username TEXT,
    first_name TEXT,
    last_name TEXT,
    language_code TEXT,
    is_bot BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Chat members (tracks users in chats)
CREATE TABLE IF NOT EXISTS chat_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chat_id BIGINT NOT NULL REFERENCES chats(chat_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    role TEXT DEFAULT 'member', -- 'owner', 'admin', 'member'
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(chat_id, user_id)
);

-- ============================================
-- MODERATION TABLES
-- ============================================

-- Bans table
CREATE TABLE IF NOT EXISTS bans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chat_id BIGINT NOT NULL REFERENCES chats(chat_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    banned_by BIGINT REFERENCES users(user_id),
    reason TEXT,
    until_date TIMESTAMPTZ, -- NULL means permanent
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(chat_id, user_id)
);

-- Mutes table
CREATE TABLE IF NOT EXISTS mutes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chat_id BIGINT NOT NULL REFERENCES chats(chat_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    muted_by BIGINT REFERENCES users(user_id),
    reason TEXT,
    until_date TIMESTAMPTZ, -- NULL means permanent
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(chat_id, user_id)
);

-- Warnings table
CREATE TABLE IF NOT EXISTS warnings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chat_id BIGINT NOT NULL REFERENCES chats(chat_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    warned_by BIGINT REFERENCES users(user_id),
    reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Warning settings per chat
CREATE TABLE IF NOT EXISTS warning_settings (
    chat_id BIGINT PRIMARY KEY REFERENCES chats(chat_id) ON DELETE CASCADE,
    warn_limit INTEGER DEFAULT 3,
    warn_mode TEXT DEFAULT 'ban', -- 'ban', 'kick', 'mute', 'tban', 'tmute'
    warn_time INTEGER DEFAULT 0, -- time in seconds for tban/tmute
    soft_warn BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- ANTI-SPAM TABLES
-- ============================================

-- Blocklists (blacklists)
CREATE TABLE IF NOT EXISTS blocklists (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chat_id BIGINT NOT NULL REFERENCES chats(chat_id) ON DELETE CASCADE,
    trigger_word TEXT NOT NULL,
    match_mode TEXT DEFAULT 'word', -- 'word', 'contains', 'regex'
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(chat_id, trigger_word)
);

-- Blocklist settings per chat
CREATE TABLE IF NOT EXISTS blocklist_settings (
    chat_id BIGINT PRIMARY KEY REFERENCES chats(chat_id) ON DELETE CASCADE,
    blocklist_mode TEXT DEFAULT 'delete', -- 'delete', 'warn', 'kick', 'ban', 'tban', 'mute', 'tmute'
    blocklist_time INTEGER DEFAULT 0, -- time for tban/tmute
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Locks table
CREATE TABLE IF NOT EXISTS locks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chat_id BIGINT NOT NULL REFERENCES chats(chat_id) ON DELETE CASCADE,
    lock_type TEXT NOT NULL, -- 'all', 'media', 'sticker', 'gif', 'url', 'forward', 'bot', 'game', 'voice', 'video', 'document', 'photo', 'audio', 'command', 'email', 'location', 'contact', 'inline', 'button', 'emoji', 'poll', 'invitelink', 'rtl', 'arabic', 'chinese', 'anonchannel'
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(chat_id, lock_type)
);

-- Lock settings per chat
CREATE TABLE IF NOT EXISTS lock_settings (
    chat_id BIGINT PRIMARY KEY REFERENCES chats(chat_id) ON DELETE CASCADE,
    lock_warn BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Antiflood settings
CREATE TABLE IF NOT EXISTS antiflood_settings (
    chat_id BIGINT PRIMARY KEY REFERENCES chats(chat_id) ON DELETE CASCADE,
    flood_limit INTEGER DEFAULT 0, -- 0 = disabled
    flood_mode TEXT DEFAULT 'ban', -- 'ban', 'kick', 'mute', 'tban', 'tmute'
    flood_time INTEGER DEFAULT 0, -- time for tban/tmute
    flood_timer INTEGER DEFAULT 0, -- seconds to track messages (0 = consecutive only)
    delete_flood_messages BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Flood tracking (temporary, for tracking user messages)
CREATE TABLE IF NOT EXISTS flood_tracking (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chat_id BIGINT NOT NULL REFERENCES chats(chat_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    message_count INTEGER DEFAULT 1,
    last_message_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(chat_id, user_id)
);

-- ============================================
-- CONTENT TABLES
-- ============================================

-- Notes table
CREATE TABLE IF NOT EXISTS notes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chat_id BIGINT NOT NULL REFERENCES chats(chat_id) ON DELETE CASCADE,
    note_name TEXT NOT NULL,
    note_content TEXT,
    note_type TEXT DEFAULT 'text', -- 'text', 'photo', 'video', 'audio', 'document', 'sticker', 'voice', 'animation'
    file_id TEXT,
    buttons JSONB, -- inline keyboard buttons
    parse_mode TEXT DEFAULT 'markdown',
    web_preview BOOLEAN DEFAULT TRUE,
    is_private BOOLEAN DEFAULT FALSE,
    created_by BIGINT REFERENCES users(user_id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(chat_id, note_name)
);

-- Filters table
CREATE TABLE IF NOT EXISTS filters (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chat_id BIGINT NOT NULL REFERENCES chats(chat_id) ON DELETE CASCADE,
    keyword TEXT NOT NULL,
    reply_text TEXT,
    reply_type TEXT DEFAULT 'text', -- same as notes
    file_id TEXT,
    buttons JSONB,
    parse_mode TEXT DEFAULT 'markdown',
    web_preview BOOLEAN DEFAULT TRUE,
    has_media BOOLEAN DEFAULT FALSE,
    created_by BIGINT REFERENCES users(user_id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(chat_id, keyword)
);

-- Welcome/Goodbye settings
CREATE TABLE IF NOT EXISTS greetings (
    chat_id BIGINT PRIMARY KEY REFERENCES chats(chat_id) ON DELETE CASCADE,
    welcome_enabled BOOLEAN DEFAULT TRUE,
    welcome_text TEXT DEFAULT 'Hey {first}, welcome to {chatname}!',
    welcome_type TEXT DEFAULT 'text',
    welcome_file_id TEXT,
    welcome_buttons JSONB,
    welcome_parse_mode TEXT DEFAULT 'markdown',
    welcome_web_preview BOOLEAN DEFAULT TRUE,
    goodbye_enabled BOOLEAN DEFAULT FALSE,
    goodbye_text TEXT DEFAULT 'Goodbye {first}!',
    goodbye_type TEXT DEFAULT 'text',
    goodbye_file_id TEXT,
    goodbye_buttons JSONB,
    goodbye_parse_mode TEXT DEFAULT 'markdown',
    clean_welcome BOOLEAN DEFAULT FALSE, -- delete previous welcome
    clean_service BOOLEAN DEFAULT FALSE, -- delete service messages
    welcome_mute BOOLEAN DEFAULT FALSE,
    welcome_mute_time INTEGER DEFAULT 0, -- mute new members for X seconds
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Rules table
CREATE TABLE IF NOT EXISTS rules (
    chat_id BIGINT PRIMARY KEY REFERENCES chats(chat_id) ON DELETE CASCADE,
    rules_text TEXT DEFAULT 'No rules have been set for this chat yet!',
    private_rules BOOLEAN DEFAULT FALSE, -- send rules in PM
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- FEDERATION TABLES
-- ============================================

-- Federations
CREATE TABLE IF NOT EXISTS federations (
    fed_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    fed_name TEXT NOT NULL,
    owner_id BIGINT NOT NULL REFERENCES users(user_id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Federation admins
CREATE TABLE IF NOT EXISTS fed_admins (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    fed_id UUID NOT NULL REFERENCES federations(fed_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    added_by BIGINT REFERENCES users(user_id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(fed_id, user_id)
);

-- Federation bans
CREATE TABLE IF NOT EXISTS fed_bans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    fed_id UUID NOT NULL REFERENCES federations(fed_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    banned_by BIGINT REFERENCES users(user_id),
    reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(fed_id, user_id)
);

-- Federation chat subscriptions
CREATE TABLE IF NOT EXISTS fed_chats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    fed_id UUID NOT NULL REFERENCES federations(fed_id) ON DELETE CASCADE,
    chat_id BIGINT NOT NULL REFERENCES chats(chat_id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(fed_id, chat_id)
);

-- ============================================
-- CONNECTIONS TABLE
-- ============================================

-- User connections (PM management of groups)
CREATE TABLE IF NOT EXISTS connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    chat_id BIGINT NOT NULL REFERENCES chats(chat_id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, chat_id)
);

-- ============================================
-- APPROVAL TABLE
-- ============================================

-- Approved users (bypass restrictions)
CREATE TABLE IF NOT EXISTS approved_users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chat_id BIGINT NOT NULL REFERENCES chats(chat_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    approved_by BIGINT REFERENCES users(user_id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(chat_id, user_id)
);

-- ============================================
-- REPORTS TABLE
-- ============================================

-- Report settings
CREATE TABLE IF NOT EXISTS report_settings (
    chat_id BIGINT PRIMARY KEY REFERENCES chats(chat_id) ON DELETE CASCADE,
    reports_enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- CAPTCHA TABLE
-- ============================================

-- Captcha settings
CREATE TABLE IF NOT EXISTS captcha_settings (
    chat_id BIGINT PRIMARY KEY REFERENCES chats(chat_id) ON DELETE CASCADE,
    captcha_enabled BOOLEAN DEFAULT FALSE,
    captcha_type TEXT DEFAULT 'button', -- 'button', 'math', 'text'
    captcha_timeout INTEGER DEFAULT 120, -- seconds
    captcha_kick BOOLEAN DEFAULT TRUE, -- kick on timeout
    captcha_text TEXT DEFAULT 'Please verify you are human by clicking the button below.',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Pending captcha verifications
CREATE TABLE IF NOT EXISTS captcha_pending (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chat_id BIGINT NOT NULL REFERENCES chats(chat_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    captcha_answer TEXT,
    message_id BIGINT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(chat_id, user_id)
);

-- ============================================
-- DISABLED COMMANDS TABLE
-- ============================================

-- Disabled commands per chat
CREATE TABLE IF NOT EXISTS disabled_commands (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chat_id BIGINT NOT NULL REFERENCES chats(chat_id) ON DELETE CASCADE,
    command TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(chat_id, command)
);

-- ============================================
-- LOG CHANNEL TABLE
-- ============================================

-- Log channel settings
CREATE TABLE IF NOT EXISTS log_channels (
    chat_id BIGINT PRIMARY KEY REFERENCES chats(chat_id) ON DELETE CASCADE,
    log_channel_id BIGINT,
    log_joins BOOLEAN DEFAULT TRUE,
    log_leaves BOOLEAN DEFAULT TRUE,
    log_kicks BOOLEAN DEFAULT TRUE,
    log_bans BOOLEAN DEFAULT TRUE,
    log_warns BOOLEAN DEFAULT TRUE,
    log_pins BOOLEAN DEFAULT TRUE,
    log_settings BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- GLOBAL BANS TABLE
-- ============================================

-- Global bans (gbans)
CREATE TABLE IF NOT EXISTS global_bans (
    user_id BIGINT PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
    banned_by BIGINT REFERENCES users(user_id),
    reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Bot owners/sudoers
CREATE TABLE IF NOT EXISTS bot_admins (
    user_id BIGINT PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
    role TEXT DEFAULT 'sudoer', -- 'owner', 'sudoer'
    added_by BIGINT REFERENCES users(user_id),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- INDEXES FOR PERFORMANCE
-- ============================================

CREATE INDEX IF NOT EXISTS idx_chat_members_chat_id ON chat_members(chat_id);
CREATE INDEX IF NOT EXISTS idx_chat_members_user_id ON chat_members(user_id);
CREATE INDEX IF NOT EXISTS idx_bans_chat_id ON bans(chat_id);
CREATE INDEX IF NOT EXISTS idx_mutes_chat_id ON mutes(chat_id);
CREATE INDEX IF NOT EXISTS idx_warnings_chat_id ON warnings(chat_id);
CREATE INDEX IF NOT EXISTS idx_warnings_user_id ON warnings(user_id);
CREATE INDEX IF NOT EXISTS idx_blocklists_chat_id ON blocklists(chat_id);
CREATE INDEX IF NOT EXISTS idx_locks_chat_id ON locks(chat_id);
CREATE INDEX IF NOT EXISTS idx_notes_chat_id ON notes(chat_id);
CREATE INDEX IF NOT EXISTS idx_filters_chat_id ON filters(chat_id);
CREATE INDEX IF NOT EXISTS idx_fed_bans_fed_id ON fed_bans(fed_id);
CREATE INDEX IF NOT EXISTS idx_fed_chats_fed_id ON fed_chats(fed_id);
CREATE INDEX IF NOT EXISTS idx_connections_user_id ON connections(user_id);
CREATE INDEX IF NOT EXISTS idx_flood_tracking_chat_user ON flood_tracking(chat_id, user_id);
