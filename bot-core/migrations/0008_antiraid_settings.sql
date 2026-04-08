CREATE TABLE IF NOT EXISTS antiraid_settings (
    bot_id TEXT NOT NULL,
    chat_id BIGINT NOT NULL,
    enabled_until TIMESTAMPTZ NULL,
    raid_duration_seconds INTEGER NOT NULL DEFAULT 21600,
    action_duration_seconds INTEGER NOT NULL DEFAULT 3600,
    auto_threshold INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, chat_id),
    FOREIGN KEY (bot_id, chat_id) REFERENCES chats(bot_id, telegram_chat_id) ON DELETE CASCADE
);
