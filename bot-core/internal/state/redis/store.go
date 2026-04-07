package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/state"
)

type Store struct {
	client *goredis.Client
}

var _ state.Store = (*Store)(nil)

func New(addr, password string, db int) *Store {
	return &Store{
		client: goredis.NewClient(&goredis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
	}
}

func (s *Store) Close() error {
	return s.client.Close()
}

func (s *Store) GetCachedBot(ctx context.Context, webhookKey string) (domain.BotInstance, bool, error) {
	value, err := s.client.Get(ctx, fmt.Sprintf("bot:webhook:%s", webhookKey)).Result()
	if err == goredis.Nil {
		return domain.BotInstance{}, false, nil
	}
	if err != nil {
		return domain.BotInstance{}, false, err
	}

	var bot domain.BotInstance
	if err := json.Unmarshal([]byte(value), &bot); err != nil {
		return domain.BotInstance{}, false, err
	}
	return bot, true, nil
}

func (s *Store) CacheBot(ctx context.Context, bot domain.BotInstance, ttl time.Duration) error {
	body, err := json.Marshal(bot)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, fmt.Sprintf("bot:webhook:%s", bot.WebhookKey), body, ttl).Err()
}

func (s *Store) TrackFlood(ctx context.Context, botID string, chatID int64, userID int64, messageID int64, window time.Duration) (int64, error) {
	key := fmt.Sprintf("flood:%s:%d:%d", botID, chatID, userID)
	now := time.Now()
	cutoff := now.Add(-window).UnixMilli()
	member := fmt.Sprintf("%d:%d", messageID, now.UnixNano())

	pipe := s.client.TxPipeline()
	pipe.ZAdd(ctx, key, goredis.Z{
		Score:  float64(now.UnixMilli()),
		Member: member,
	})
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", cutoff))
	countCmd := pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, window+time.Minute)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return countCmd.Val(), nil
}

func (s *Store) ClearFlood(ctx context.Context, botID string, chatID int64, userID int64) error {
	return s.client.Del(ctx, fmt.Sprintf("flood:%s:%d:%d", botID, chatID, userID)).Err()
}

func (s *Store) AcquireLease(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return s.client.SetNX(ctx, "lease:"+key, "1", ttl).Result()
}

func (s *Store) SetLease(ctx context.Context, key string, ttl time.Duration) error {
	return s.client.Set(ctx, "lease:"+key, "1", ttl).Err()
}

func (s *Store) DeleteLease(ctx context.Context, key string) error {
	return s.client.Del(ctx, "lease:"+key).Err()
}
