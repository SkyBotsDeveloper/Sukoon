package state

import (
	"context"
	"time"

	"sukoon/bot-core/internal/domain"
)

type Store interface {
	Close() error
	GetCachedBot(ctx context.Context, webhookKey string) (domain.BotInstance, bool, error)
	CacheBot(ctx context.Context, bot domain.BotInstance, ttl time.Duration) error
	TrackFlood(ctx context.Context, botID string, chatID int64, userID int64, messageID int64, window time.Duration) (int64, error)
	ClearFlood(ctx context.Context, botID string, chatID int64, userID int64) error
	AcquireLease(ctx context.Context, key string, ttl time.Duration) (bool, error)
}
