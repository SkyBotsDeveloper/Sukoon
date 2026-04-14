CREATE TABLE IF NOT EXISTS chat_connection_history (
    bot_id TEXT NOT NULL,
    user_id BIGINT NOT NULL,
    chat_id BIGINT NOT NULL,
    connected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, user_id, chat_id),
    FOREIGN KEY (bot_id, chat_id) REFERENCES chats(bot_id, telegram_chat_id) ON DELETE CASCADE
);

INSERT INTO chat_connection_history (bot_id, user_id, chat_id, connected_at)
SELECT bot_id, user_id, chat_id, connected_at
FROM chat_connections
ON CONFLICT (bot_id, user_id, chat_id) DO UPDATE SET
    connected_at = GREATEST(chat_connection_history.connected_at, EXCLUDED.connected_at);
