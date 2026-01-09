-- Complete schema fix for Sukoon Bot
-- Run this script to add missing tables and fix issues

-- ============================================
-- DROP FOREIGN KEY CONSTRAINTS (Allow flexibility)
-- ============================================

-- Drop existing foreign key constraints to make operations easier
ALTER TABLE IF EXISTS chat_members DROP CONSTRAINT IF EXISTS chat_members_chat_id_fkey;
ALTER TABLE IF EXISTS chat_members DROP CONSTRAINT IF EXISTS chat_members_user_id_fkey;
ALTER TABLE IF EXISTS bans DROP CONSTRAINT IF EXISTS bans_chat_id_fkey;
ALTER TABLE IF EXISTS bans DROP CONSTRAINT IF EXISTS bans_user_id_fkey;
ALTER TABLE IF EXISTS bans DROP CONSTRAINT IF EXISTS bans_banned_by_fkey;
ALTER TABLE IF EXISTS mutes DROP CONSTRAINT IF EXISTS mutes_chat_id_fkey;
ALTER TABLE IF EXISTS mutes DROP CONSTRAINT IF EXISTS mutes_user_id_fkey;
ALTER TABLE IF EXISTS mutes DROP CONSTRAINT IF EXISTS mutes_muted_by_fkey;
ALTER TABLE IF EXISTS warnings DROP CONSTRAINT IF EXISTS warnings_chat_id_fkey;
ALTER TABLE IF EXISTS warnings DROP CONSTRAINT IF EXISTS warnings_user_id_fkey;
ALTER TABLE IF EXISTS warnings DROP CONSTRAINT IF EXISTS warnings_warned_by_fkey;
ALTER TABLE IF EXISTS warning_settings DROP CONSTRAINT IF EXISTS warning_settings_chat_id_fkey;
ALTER TABLE IF EXISTS blocklists DROP CONSTRAINT IF EXISTS blocklists_chat_id_fkey;
ALTER TABLE IF EXISTS blocklist_settings DROP CONSTRAINT IF EXISTS blocklist_settings_chat_id_fkey;
ALTER TABLE IF EXISTS locks DROP CONSTRAINT IF EXISTS locks_chat_id_fkey;
ALTER TABLE IF EXISTS lock_settings DROP CONSTRAINT IF EXISTS lock_settings_chat_id_fkey;
ALTER TABLE IF EXISTS antiflood_settings DROP CONSTRAINT IF EXISTS antiflood_settings_chat_id_fkey;
ALTER TABLE IF EXISTS flood_tracking DROP CONSTRAINT IF EXISTS flood_tracking_chat_id_fkey;
ALTER TABLE IF EXISTS flood_tracking DROP CONSTRAINT IF EXISTS flood_tracking_user_id_fkey;
ALTER TABLE IF EXISTS notes DROP CONSTRAINT IF EXISTS notes_chat_id_fkey;
ALTER TABLE IF EXISTS notes DROP CONSTRAINT IF EXISTS notes_created_by_fkey;
ALTER TABLE IF EXISTS filters DROP CONSTRAINT IF EXISTS filters_chat_id_fkey;
ALTER TABLE IF EXISTS filters DROP CONSTRAINT IF EXISTS filters_created_by_fkey;
ALTER TABLE IF EXISTS greetings DROP CONSTRAINT IF EXISTS greetings_chat_id_fkey;
ALTER TABLE IF EXISTS rules DROP CONSTRAINT IF EXISTS rules_chat_id_fkey;
ALTER TABLE IF EXISTS fed_admins DROP CONSTRAINT IF EXISTS fed_admins_fed_id_fkey;
ALTER TABLE IF EXISTS fed_admins DROP CONSTRAINT IF EXISTS fed_admins_user_id_fkey;
ALTER TABLE IF EXISTS fed_admins DROP CONSTRAINT IF EXISTS fed_admins_added_by_fkey;
ALTER TABLE IF EXISTS fed_bans DROP CONSTRAINT IF EXISTS fed_bans_fed_id_fkey;
ALTER TABLE IF EXISTS fed_bans DROP CONSTRAINT IF EXISTS fed_bans_user_id_fkey;
ALTER TABLE IF EXISTS fed_bans DROP CONSTRAINT IF EXISTS fed_bans_banned_by_fkey;
ALTER TABLE IF EXISTS fed_chats DROP CONSTRAINT IF EXISTS fed_chats_fed_id_fkey;
ALTER TABLE IF EXISTS fed_chats DROP CONSTRAINT IF EXISTS fed_chats_chat_id_fkey;
ALTER TABLE IF EXISTS connections DROP CONSTRAINT IF EXISTS connections_user_id_fkey;
ALTER TABLE IF EXISTS connections DROP CONSTRAINT IF EXISTS connections_chat_id_fkey;
ALTER TABLE IF EXISTS approved_users DROP CONSTRAINT IF EXISTS approved_users_chat_id_fkey;
ALTER TABLE IF EXISTS approved_users DROP CONSTRAINT IF EXISTS approved_users_user_id_fkey;
ALTER TABLE IF EXISTS approved_users DROP CONSTRAINT IF EXISTS approved_users_approved_by_fkey;
ALTER TABLE IF EXISTS report_settings DROP CONSTRAINT IF EXISTS report_settings_chat_id_fkey;
ALTER TABLE IF EXISTS captcha_settings DROP CONSTRAINT IF EXISTS captcha_settings_chat_id_fkey;
ALTER TABLE IF EXISTS captcha_pending DROP CONSTRAINT IF EXISTS captcha_pending_chat_id_fkey;
ALTER TABLE IF EXISTS captcha_pending DROP CONSTRAINT IF EXISTS captcha_pending_user_id_fkey;
ALTER TABLE IF EXISTS disabled_commands DROP CONSTRAINT IF EXISTS disabled_commands_chat_id_fkey;
ALTER TABLE IF EXISTS log_channels DROP CONSTRAINT IF EXISTS log_channels_chat_id_fkey;
ALTER TABLE IF EXISTS global_bans DROP CONSTRAINT IF EXISTS global_bans_user_id_fkey;
ALTER TABLE IF EXISTS global_bans DROP CONSTRAINT IF EXISTS global_bans_banned_by_fkey;
ALTER TABLE IF EXISTS bot_admins DROP CONSTRAINT IF EXISTS bot_admins_user_id_fkey;
ALTER TABLE IF EXISTS bot_admins DROP CONSTRAINT IF EXISTS bot_admins_added_by_fkey;
ALTER TABLE IF EXISTS federations DROP CONSTRAINT IF EXISTS federations_owner_id_fkey;

-- ============================================
-- ADD MISSING TABLES
-- ============================================

-- AntiRaid settings table
CREATE TABLE IF NOT EXISTS antiraid_settings (
    chat_id BIGINT PRIMARY KEY,
    antiraid_enabled BOOLEAN DEFAULT FALSE,
    raid_duration INTEGER DEFAULT 21600, -- 6 hours default
    raid_action_time INTEGER DEFAULT 3600, -- 1 hour ban default
    auto_raid_threshold INTEGER DEFAULT 0, -- 0 = disabled
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Clean Commands settings table
CREATE TABLE IF NOT EXISTS clean_commands_settings (
    chat_id BIGINT PRIMARY KEY,
    clean_all BOOLEAN DEFAULT FALSE,
    clean_admin BOOLEAN DEFAULT FALSE,
    clean_user BOOLEAN DEFAULT FALSE,
    clean_other BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Clean Service settings table
CREATE TABLE IF NOT EXISTS clean_service_settings (
    chat_id BIGINT PRIMARY KEY,
    clean_service_enabled BOOLEAN DEFAULT FALSE,
    clean_all BOOLEAN DEFAULT FALSE,
    clean_join BOOLEAN DEFAULT FALSE,
    clean_leave BOOLEAN DEFAULT FALSE,
    clean_pin BOOLEAN DEFAULT FALSE,
    clean_title BOOLEAN DEFAULT FALSE,
    clean_photo BOOLEAN DEFAULT FALSE,
    clean_video_chat BOOLEAN DEFAULT FALSE,
    clean_other BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Chat settings table (for rules and other settings)
CREATE TABLE IF NOT EXISTS chat_settings (
    chat_id BIGINT PRIMARY KEY,
    rules TEXT,
    rules_button TEXT DEFAULT 'Rules',
    private_rules BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Fed subscriptions table (for subscribing to other feds)
CREATE TABLE IF NOT EXISTS fed_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fed_id UUID NOT NULL,
    subscribed_fed_id UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(fed_id, subscribed_fed_id)
);

-- Fed settings table
CREATE TABLE IF NOT EXISTS fed_settings (
    fed_id UUID PRIMARY KEY,
    quiet_fed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- CREATE INDEXES
-- ============================================

CREATE INDEX IF NOT EXISTS idx_antiraid_chat_id ON antiraid_settings(chat_id);
CREATE INDEX IF NOT EXISTS idx_clean_commands_chat_id ON clean_commands_settings(chat_id);
CREATE INDEX IF NOT EXISTS idx_clean_service_chat_id ON clean_service_settings(chat_id);
CREATE INDEX IF NOT EXISTS idx_chat_settings_chat_id ON chat_settings(chat_id);

-- ============================================
-- INSERT DEFAULT BOT ADMIN (your user ID)
-- ============================================

-- You can set yourself as bot admin by running:
-- INSERT INTO users (user_id, first_name) VALUES (YOUR_USER_ID, 'Your Name') ON CONFLICT (user_id) DO NOTHING;
-- INSERT INTO bot_admins (user_id, role) VALUES (YOUR_USER_ID, 'owner') ON CONFLICT (user_id) DO NOTHING;
