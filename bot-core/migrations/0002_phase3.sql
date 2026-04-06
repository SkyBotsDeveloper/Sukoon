ALTER TABLE bot_instances
    ADD COLUMN IF NOT EXISTS created_by_user_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_unique
    ON users (LOWER(username))
    WHERE username <> '';

CREATE TABLE IF NOT EXISTS chat_roles (
    bot_id TEXT NOT NULL REFERENCES bot_instances(id) ON DELETE CASCADE,
    chat_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    role TEXT NOT NULL,
    granted_by BIGINT NOT NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, chat_id, user_id, role),
    FOREIGN KEY (bot_id, chat_id) REFERENCES chats(bot_id, telegram_chat_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS antiabuse_settings (
    bot_id TEXT NOT NULL,
    chat_id BIGINT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    action TEXT NOT NULL DEFAULT 'delete_warn',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, chat_id),
    FOREIGN KEY (bot_id, chat_id) REFERENCES chats(bot_id, telegram_chat_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS antibio_settings (
    bot_id TEXT NOT NULL,
    chat_id BIGINT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    action TEXT NOT NULL DEFAULT 'kick',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, chat_id),
    FOREIGN KEY (bot_id, chat_id) REFERENCES chats(bot_id, telegram_chat_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS antibio_exemptions (
    bot_id TEXT NOT NULL,
    chat_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    added_by BIGINT NOT NULL,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bot_id, chat_id, user_id),
    FOREIGN KEY (bot_id, chat_id) REFERENCES chats(bot_id, telegram_chat_id) ON DELETE CASCADE
);

ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS requested_by BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS report_chat_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS attempts INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_attempts INTEGER NOT NULL DEFAULT 5,
    ADD COLUMN IF NOT EXISTS available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS locked_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS locked_by TEXT NULL,
    ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS result_summary TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_jobs_claim
    ON jobs (status, available_at, created_at);
