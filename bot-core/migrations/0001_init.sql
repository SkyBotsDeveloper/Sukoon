CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS bot_instances (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    telegram_token TEXT NOT NULL,
    webhook_key TEXT NOT NULL UNIQUE,
    webhook_secret TEXT NOT NULL,
    username TEXT NOT NULL DEFAULT '',
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS bot_roles (
    bot_id TEXT NOT NULL REFERENCES bot_instances(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    role TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, user_id, role)
);

CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY,
    username TEXT NOT NULL DEFAULT '',
    first_name TEXT NOT NULL DEFAULT '',
    last_name TEXT NOT NULL DEFAULT '',
    is_bot BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS chats (
    bot_id TEXT NOT NULL REFERENCES bot_instances(id) ON DELETE CASCADE,
    telegram_chat_id BIGINT NOT NULL,
    chat_type TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    username TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, telegram_chat_id)
);

CREATE TABLE IF NOT EXISTS chat_settings (
    bot_id TEXT NOT NULL,
    chat_id BIGINT NOT NULL,
    language TEXT NOT NULL DEFAULT 'en',
    reports_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    log_channel_id BIGINT NULL,
    clean_commands BOOLEAN NOT NULL DEFAULT FALSE,
    clean_service_join BOOLEAN NOT NULL DEFAULT FALSE,
    clean_service_leave BOOLEAN NOT NULL DEFAULT FALSE,
    clean_service_pin BOOLEAN NOT NULL DEFAULT FALSE,
    welcome_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    welcome_text TEXT NOT NULL DEFAULT '',
    goodbye_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    goodbye_text TEXT NOT NULL DEFAULT '',
    rules_text TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, chat_id),
    FOREIGN KEY (bot_id, chat_id) REFERENCES chats(bot_id, telegram_chat_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS moderation_settings (
    bot_id TEXT NOT NULL,
    chat_id BIGINT NOT NULL,
    warn_limit INTEGER NOT NULL DEFAULT 3,
    warn_mode TEXT NOT NULL DEFAULT 'mute',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, chat_id),
    FOREIGN KEY (bot_id, chat_id) REFERENCES chats(bot_id, telegram_chat_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS warnings (
    bot_id TEXT NOT NULL,
    chat_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    warning_count INTEGER NOT NULL DEFAULT 0,
    last_reason TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, chat_id, user_id),
    FOREIGN KEY (bot_id, chat_id) REFERENCES chats(bot_id, telegram_chat_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS approvals (
    bot_id TEXT NOT NULL,
    chat_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    approved_by BIGINT NOT NULL,
    approved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, chat_id, user_id),
    FOREIGN KEY (bot_id, chat_id) REFERENCES chats(bot_id, telegram_chat_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS disabled_commands (
    bot_id TEXT NOT NULL,
    chat_id BIGINT NOT NULL,
    command TEXT NOT NULL,
    changed_by BIGINT NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, chat_id, command),
    FOREIGN KEY (bot_id, chat_id) REFERENCES chats(bot_id, telegram_chat_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS locks (
    bot_id TEXT NOT NULL,
    chat_id BIGINT NOT NULL,
    lock_type TEXT NOT NULL,
    action TEXT NOT NULL DEFAULT 'delete',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, chat_id, lock_type),
    FOREIGN KEY (bot_id, chat_id) REFERENCES chats(bot_id, telegram_chat_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS blocklist_rules (
    id BIGSERIAL PRIMARY KEY,
    bot_id TEXT NOT NULL,
    chat_id BIGINT NOT NULL,
    pattern TEXT NOT NULL,
    match_mode TEXT NOT NULL DEFAULT 'word',
    action TEXT NOT NULL DEFAULT 'delete',
    created_by BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (bot_id, chat_id, pattern, match_mode),
    FOREIGN KEY (bot_id, chat_id) REFERENCES chats(bot_id, telegram_chat_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS antiflood_settings (
    bot_id TEXT NOT NULL,
    chat_id BIGINT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    flood_limit INTEGER NOT NULL DEFAULT 6,
    window_seconds INTEGER NOT NULL DEFAULT 10,
    action TEXT NOT NULL DEFAULT 'mute',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, chat_id),
    FOREIGN KEY (bot_id, chat_id) REFERENCES chats(bot_id, telegram_chat_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS notes (
    bot_id TEXT NOT NULL,
    chat_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    text TEXT NOT NULL,
    parse_mode TEXT NOT NULL DEFAULT '',
    buttons_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_by BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, chat_id, name),
    FOREIGN KEY (bot_id, chat_id) REFERENCES chats(bot_id, telegram_chat_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS filters (
    id BIGSERIAL PRIMARY KEY,
    bot_id TEXT NOT NULL,
    chat_id BIGINT NOT NULL,
    trigger TEXT NOT NULL,
    match_mode TEXT NOT NULL DEFAULT 'contains',
    response_text TEXT NOT NULL,
    parse_mode TEXT NOT NULL DEFAULT '',
    buttons_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_by BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (bot_id, chat_id, trigger),
    FOREIGN KEY (bot_id, chat_id) REFERENCES chats(bot_id, telegram_chat_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS captcha_settings (
    bot_id TEXT NOT NULL,
    chat_id BIGINT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    mode TEXT NOT NULL DEFAULT 'button',
    timeout_seconds INTEGER NOT NULL DEFAULT 120,
    failure_action TEXT NOT NULL DEFAULT 'kick',
    challenge_digits INTEGER NOT NULL DEFAULT 2,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, chat_id),
    FOREIGN KEY (bot_id, chat_id) REFERENCES chats(bot_id, telegram_chat_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS captcha_challenges (
    id TEXT PRIMARY KEY,
    bot_id TEXT NOT NULL,
    chat_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    prompt TEXT NOT NULL,
    answer TEXT NOT NULL,
    message_id BIGINT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    failure_action TEXT NOT NULL DEFAULT 'kick',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (bot_id, chat_id) REFERENCES chats(bot_id, telegram_chat_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS chat_connections (
    bot_id TEXT NOT NULL,
    user_id BIGINT NOT NULL,
    chat_id BIGINT NOT NULL,
    connected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, user_id),
    FOREIGN KEY (bot_id, chat_id) REFERENCES chats(bot_id, telegram_chat_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS afk_states (
    bot_id TEXT NOT NULL,
    user_id BIGINT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    set_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, user_id),
    FOREIGN KEY (bot_id) REFERENCES bot_instances(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS federations (
    id TEXT PRIMARY KEY,
    bot_id TEXT NOT NULL REFERENCES bot_instances(id) ON DELETE CASCADE,
    short_name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    owner_user_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (bot_id, short_name)
);

CREATE TABLE IF NOT EXISTS federation_chats (
    federation_id TEXT NOT NULL REFERENCES federations(id) ON DELETE CASCADE,
    bot_id TEXT NOT NULL REFERENCES bot_instances(id) ON DELETE CASCADE,
    chat_id BIGINT NOT NULL,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (federation_id, bot_id, chat_id)
);

CREATE TABLE IF NOT EXISTS federation_admins (
    federation_id TEXT NOT NULL REFERENCES federations(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    role TEXT NOT NULL DEFAULT 'admin',
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (federation_id, user_id)
);

CREATE TABLE IF NOT EXISTS federation_bans (
    federation_id TEXT NOT NULL REFERENCES federations(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    banned_by BIGINT NOT NULL,
    banned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (federation_id, user_id)
);

CREATE TABLE IF NOT EXISTS global_blacklist_users (
    bot_id TEXT NOT NULL REFERENCES bot_instances(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    created_by BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, user_id)
);

CREATE TABLE IF NOT EXISTS global_blacklist_chats (
    bot_id TEXT NOT NULL REFERENCES bot_instances(id) ON DELETE CASCADE,
    chat_id BIGINT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    created_by BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, chat_id)
);

CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    bot_id TEXT NOT NULL REFERENCES bot_instances(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    progress INTEGER NOT NULL DEFAULT 0,
    total INTEGER NOT NULL DEFAULT 0,
    payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_text TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS telegram_updates (
    id BIGSERIAL PRIMARY KEY,
    bot_id TEXT NOT NULL REFERENCES bot_instances(id) ON DELETE CASCADE,
    update_id BIGINT NOT NULL,
    payload_json JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_at TIMESTAMPTZ NULL,
    locked_by TEXT NULL,
    processed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (bot_id, update_id)
);

CREATE INDEX IF NOT EXISTS idx_telegram_updates_claim
    ON telegram_updates (status, available_at, created_at);

CREATE INDEX IF NOT EXISTS idx_captcha_challenges_expiry
    ON captcha_challenges (status, expires_at);

CREATE UNIQUE INDEX IF NOT EXISTS idx_captcha_pending_unique
    ON captcha_challenges (bot_id, chat_id, user_id)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_filters_chat
    ON filters (bot_id, chat_id);

CREATE INDEX IF NOT EXISTS idx_blocklist_chat
    ON blocklist_rules (bot_id, chat_id);
