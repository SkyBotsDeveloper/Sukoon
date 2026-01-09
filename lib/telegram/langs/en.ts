// English language file - Default

// Define the translations object
const translations = {
  // General
  lang_name: "English",
  lang_code: "en",

  // Start & Help
  start_welcome:
    "Hey there! My name is {bot_name} - I'm here to help you manage your groups! Use /help to find out how to use me to my full potential.\n\nJoin my news channel to get information on all the latest updates.\n\nCheck /privacy to view the privacy policy, and interact with your data.",
  start_welcome_group: "Hey! I'm {bot_name}, your group management assistant. Use /help to see what I can do!",
  help_header:
    "Hey! My name is {bot_name}. I am a group management bot, here to help you get around and keep the order in your groups!\n\nI have lots of handy features, such as flood control, a warning system, a note keeping system, and even predetermined replies on certain keywords.\n\nHelpful commands:\n- /start: Starts me! You've probably already used this.\n- /help: Sends this message; I'll tell you more about myself!\n- /donate: Gives you info on how to support me and my creator.\n\nIf you have any bugs or questions on how to use me, have a look at my website, or head to @{support_chat}.\n\nAll commands can be used with the following: / !",

  // Buttons
  btn_add_to_chat: "Add me to your chat!",
  btn_get_own: "Get your own {bot_name}",
  btn_back: "« Back",
  btn_support: "Support",
  btn_updates: "Updates Channel",

  // Admin commands
  admin_promote_success: "Successfully promoted {user} in {chat}!",
  admin_promote_fail: "Failed to promote user: {error}",
  admin_demote_success: "Successfully demoted {user} in {chat}!",
  admin_demote_fail: "Failed to demote user: {error}",
  admin_no_reply: "Please reply to a user or provide a user ID/username.",
  admin_not_admin: "You need to be an admin to use this command.",
  admin_bot_not_admin: "I need to be an admin to do this!",
  admin_cant_self: "You can't do this to yourself!",
  admin_cant_owner: "I can't do this to the chat owner!",
  admin_cant_admin: "I can't do this to another admin!",
  admin_list_title: "Admins in {chat}:",
  admin_list_creator: "Creator",
  admin_list_admin: "Admin",

  // Moderation - Ban
  ban_success: "Banned {user}!\nReason: {reason}",
  ban_success_no_reason: "Banned {user}!",
  ban_fail: "Failed to ban user: {error}",
  unban_success: "Unbanned {user}!",
  unban_fail: "Failed to unban user: {error}",

  // Moderation - Mute
  mute_success: "Muted {user}!\nDuration: {duration}\nReason: {reason}",
  mute_success_no_reason: "Muted {user}!\nDuration: {duration}",
  mute_success_permanent: "Muted {user} permanently!\nReason: {reason}",
  mute_fail: "Failed to mute user: {error}",
  unmute_success: "Unmuted {user}!",
  unmute_fail: "Failed to unmute user: {error}",

  // Moderation - Kick
  kick_success: "Kicked {user}!\nReason: {reason}",
  kick_success_no_reason: "Kicked {user}!",
  kick_fail: "Failed to kick user: {error}",

  // Moderation - Warn
  warn_success: "Warned {user} ({count}/{max})!\nReason: {reason}",
  warn_limit_reached: "User {user} has reached the warn limit ({max})!\nAction: {action}",
  warn_removed: "Removed one warning from {user}. They now have {count} warnings.",
  warn_reset: "Warnings reset for {user}!",
  warns_list: "Warnings for {user}:\n{list}",
  warns_none: "{user} has no warnings.",

  // Approvals
  approve_success: "Approved {user}! They will now be ignored by automated admin actions.",
  approve_already: "{user} is already approved!",
  disapprove_success: "Disapproved {user}! They will now be subject to automated admin actions.",
  disapprove_not: "{user} is not approved!",
  approved_list: "Approved users in {chat}:\n{list}",
  approved_none: "No approved users in this chat.",

  // Notes
  note_saved: "Saved note '{name}'!",
  note_deleted: "Deleted note '{name}'!",
  note_not_found: "Note '{name}' not found!",
  notes_list: "Notes in {chat}:\n{list}",
  notes_none: "No notes in this chat.",

  // Filters
  filter_added: "Added filter for '{trigger}'!",
  filter_deleted: "Deleted filter '{trigger}'!",
  filter_not_found: "Filter '{trigger}' not found!",
  filters_list: "Filters in {chat}:\n{list}",
  filters_none: "No filters in this chat.",

  // Welcome
  welcome_set: "Welcome message set!",
  welcome_reset: "Welcome message reset to default!",
  welcome_current: "Current welcome message:\n{message}",
  welcome_on: "Welcome messages enabled!",
  welcome_off: "Welcome messages disabled!",
  goodbye_set: "Goodbye message set!",
  goodbye_reset: "Goodbye message reset to default!",
  goodbye_on: "Goodbye messages enabled!",
  goodbye_off: "Goodbye messages disabled!",
  default_welcome: "Welcome to {chat}, {user}! Please read the /rules.",
  default_goodbye: "Goodbye {user}! We'll miss you.",

  // Rules
  rules_set: "Rules set for this chat!",
  rules_clear: "Rules cleared!",
  rules_not_set: "No rules set for this chat!",
  rules_title: "Rules for {chat}:",

  // Locks
  lock_success: "Locked {type}!",
  unlock_success: "Unlocked {type}!",
  lock_invalid: "Invalid lock type. Valid types: {types}",
  locks_list: "Current locks in {chat}:\n{list}",
  lock_message: "This type of message is not allowed here!",

  // Antiflood
  flood_set: "Flood limit set to {count} messages in {time} seconds.",
  flood_off: "Antiflood disabled.",
  flood_action: "Flood action set to {action}.",
  flood_triggered: "{user} has been {action} for flooding!",

  // Blocklist
  blocklist_added: "Added '{word}' to blocklist!",
  blocklist_removed: "Removed '{word}' from blocklist!",
  blocklist_list: "Blocklisted words in {chat}:\n{list}",
  blocklist_none: "No blocklisted words in this chat.",
  blocklist_triggered: "Message contained blocklisted word!",

  // Federation
  fed_created:
    "Created federation '{name}'!\n\nFederation ID: `{id}`\n\nUse this ID to have other groups join your federation.",
  fed_create_fail: "Failed to create federation: {error}",
  fed_exists: "You already own a federation! Use /delfed to delete it first.",
  fed_joined: "This chat has joined federation: {name}\n\nFederation bans will now apply to this chat.",
  fed_join_fail: "Failed to join federation: {error}",
  fed_not_found: "Federation not found!",
  fed_already_joined: "This chat is already in a federation!",
  fed_left: "This chat has left federation: {name}",
  fed_not_in: "This chat is not in any federation.",
  fed_banned: "Federation banned {user}!\nFederation: {fed}\nReason: {reason}",
  fed_unbanned: "Federation unbanned {user} from {fed}!",
  fed_promoted: "Promoted {user} to federation admin in {fed}!",
  fed_demoted: "Demoted {user} from federation admin in {fed}!",
  fed_info:
    "Federation Info:\n\nName: {name}\nID: `{id}`\nOwner: {owner}\nChats: {chats}\nBans: {bans}\nAdmins: {admins}",
  fed_chat_info: "This chat is connected to:\n\nFederation: {name}\nID: `{id}`\nOwner: {owner}",
  fed_admins_list: "Federation admins in {fed}:\n{list}",
  fed_not_admin: "You are not a federation admin!",
  fed_not_owner: "You are not the federation owner!",
  fed_deleted: "Federation '{name}' has been deleted!",
  fed_user_banned: "This user is federation banned!\nReason: {reason}\nFederation: {fed}",

  // AntiRaid
  antiraid_on: "AntiRaid enabled! New members will be {action}.",
  antiraid_off: "AntiRaid disabled!",
  antiraid_action: "AntiRaid action set to {action}.",
  antiraid_triggered: "AntiRaid: {user} has been {action}!",

  // CAPTCHA
  captcha_on: "CAPTCHA verification enabled!",
  captcha_off: "CAPTCHA verification disabled!",
  captcha_mode: "CAPTCHA mode set to {mode}.",
  captcha_prompt: "Welcome {user}! Please verify you're human by clicking the button below.",
  captcha_success: "Verification successful! Welcome to {chat}!",
  captcha_fail: "Verification failed! You have been removed.",
  captcha_timeout: "{user} has been removed for not completing CAPTCHA verification.",

  // Clean Commands
  cleancommands_on: "Command deletion enabled! Bot commands will be deleted.",
  cleancommands_off: "Command deletion disabled!",

  // Clean Service
  cleanservice_on: "Service message deletion enabled!",
  cleanservice_off: "Service message deletion disabled!",

  // Reports
  report_sent: "Report sent to admins!",
  reports_on: "Reporting enabled!",
  reports_off: "Reporting disabled!",

  // Pin
  pin_success: "Message pinned!",
  pin_fail: "Failed to pin message: {error}",
  unpin_success: "Message unpinned!",
  unpin_all: "All messages unpinned!",

  // Purge
  purge_success: "Deleted {count} messages!",
  purge_fail: "Failed to purge messages: {error}",

  // User Info
  info_title: "User Info",
  info_id: "ID: {id}",
  info_first_name: "First Name: {name}",
  info_last_name: "Last Name: {name}",
  info_username: "Username: @{username}",
  info_link: "User Link: {link}",
  info_admin: "Is Admin: {status}",
  info_warns: "Warnings: {count}",
  info_approved: "Approved: {status}",
  info_gbanned: "Globally Banned: {status}",

  // Language
  lang_changed: "Language changed to {lang}!",
  lang_current: "Current language: {lang}",
  lang_list: "Available languages:\n{list}",
  lang_invalid: "Invalid language code! Use /language to see available languages.",

  // Privacy
  privacy_title: "Privacy Policy",
  privacy_text:
    "I store the following data:\n- User IDs for warns, approvals, and bans\n- Chat IDs for settings\n- Message content for notes and filters\n\nUse /gdpr to request your data.\nUse /deldata to delete your data.",

  // Errors
  error_generic: "An error occurred: {error}",
  error_no_permission: "You don't have permission to do this!",
  error_bot_no_permission: "I don't have permission to do this!",
  error_user_not_found: "User not found!",
  error_chat_not_found: "Chat not found!",
  error_invalid_args: "Invalid arguments! Usage: {usage}",
  error_group_only: "This command can only be used in groups!",
  error_private_only: "This command can only be used in private chat!",
  error_admin_only: "This command can only be used by admins!",
  error_owner_only: "This command can only be used by the chat owner!",

  // Misc
  id_info: "Chat ID: {chat_id}\nYour ID: {user_id}",
  ping_response: "Pong! {time}ms",
  stats_title: "Bot Statistics",
  stats_users: "Users: {count}",
  stats_chats: "Chats: {count}",
  stats_uptime: "Uptime: {time}",
  donate_text: "If you'd like to support the development of {bot_name}, you can donate here:\n{link}",

  // Connections
  connection_success: "Connected to {chat}! You can now manage it from PM.",
  connection_fail: "Failed to connect: {error}",
  connection_none: "You're not connected to any chat.",
  connection_current: "Currently connected to: {chat}",
  disconnect_success: "Disconnected from {chat}!",

  // Log Channel
  log_set: "Log channel set to {channel}!",
  log_unset: "Log channel removed!",
  log_current: "Current log channel: {channel}",
  log_none: "No log channel set.",

  // Disabling
  disable_success: "Disabled command {cmd}!",
  enable_success: "Enabled command {cmd}!",
  disabled_list: "Disabled commands in {chat}:\n{list}",
  disabled_none: "No disabled commands in this chat.",
  cmd_disabled: "This command is disabled in this chat!",

  // Clone
  clone_start: "Creating your own {bot_name} instance...",
  clone_validating: "Validating bot token...",
  clone_setting_webhook: "Setting up webhook...",
  clone_success:
    "Your {bot_name} clone is ready!\n\nBot: @{username}\n\nYou now have full control over your own instance. All features work exactly like the main bot.",
  clone_fail: "Failed to create clone: {error}",
  clone_invalid_token: "Invalid bot token format! Get a token from BotFather.",
  clone_instructions:
    "To create your own instance of {bot_name}:\n\n1. Create a new bot with BotFather\n2. Copy the bot token\n3. Send /clone <token> in PM\n\nYour clone will have all features of {bot_name}!",
}

// Export translations
export const en = translations

// Export type for translation keys
export type TranslationKey = keyof typeof translations
