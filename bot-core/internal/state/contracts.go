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
	TrackFlood(ctx context.Context, botID string, chatID int64, userID int64, messageID int64, window time.Duration) (domain.FloodTrackResult, error)
	ClearFlood(ctx context.Context, botID string, chatID int64, userID int64) error
	AcquireLease(ctx context.Context, key string, ttl time.Duration) (bool, error)
	SetLease(ctx context.Context, key string, ttl time.Duration) error
	DeleteLease(ctx context.Context, key string) error
}
