ALTER TABLE federations
    ADD COLUMN IF NOT EXISTS notify_actions BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS require_reason BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS log_chat_id BIGINT,
    ADD COLUMN IF NOT EXISTS log_language TEXT NOT NULL DEFAULT 'en';

ALTER TABLE federation_chats
    ADD COLUMN IF NOT EXISTS quiet_fed BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS federation_subscriptions (
    federation_id TEXT NOT NULL REFERENCES federations(id) ON DELETE CASCADE,
    subscribed_federation_id TEXT NOT NULL REFERENCES federations(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (federation_id, subscribed_federation_id),
    CHECK (federation_id <> subscribed_federation_id)
);

CREATE INDEX IF NOT EXISTS federation_subscriptions_subscribed_idx
    ON federation_subscriptions(subscribed_federation_id);
