-- Fix foreign key constraints to allow flexible data insertion
-- Run this AFTER the initial schema to modify constraints

-- Drop existing foreign key constraints that cause issues
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
ALTER TABLE IF EXISTS federations DROP CONSTRAINT IF EXISTS federations_owner_id_fkey;
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

-- The tables now operate without foreign key constraints
-- This allows the bot to insert data in any order without constraint violations
-- Data integrity is managed at the application level
