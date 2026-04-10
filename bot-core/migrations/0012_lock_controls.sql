ALTER TABLE chat_settings
    ADD COLUMN IF NOT EXISTS lock_warns BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE locks
    ADD COLUMN IF NOT EXISTS action_duration_seconds INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS reason TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS lock_allowlist (
    bot_id TEXT NOT NULL,
    chat_id BIGINT NOT NULL,
    item TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, chat_id, item),
    FOREIGN KEY (bot_id, chat_id) REFERENCES chats(bot_id, telegram_chat_id) ON DELETE CASCADE
);
