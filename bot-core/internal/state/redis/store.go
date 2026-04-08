package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
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

func (s *Store) TrackFlood(ctx context.Context, botID string, chatID int64, userID int64, messageID int64, window time.Duration) (domain.FloodTrackResult, error) {
	timedKey := fmt.Sprintf("flood:timed:%s:%d:%d", botID, chatID, userID)
	streakKey := fmt.Sprintf("flood:streak:messages:%s:%d", botID, chatID)
	lastSenderKey := fmt.Sprintf("flood:streak:last:%s:%d", botID, chatID)
	now := time.Now()
	cutoff := now.Add(-window).UnixMilli()
	member := fmt.Sprintf("%d:%d", messageID, now.UnixNano())
	streakTTL := window
	if streakTTL < 30*time.Minute {
		streakTTL = 30 * time.Minute
	}

	pipe := s.client.TxPipeline()
	pipe.ZAdd(ctx, timedKey, goredis.Z{
		Score:  float64(now.UnixMilli()),
		Member: member,
	})
	pipe.ZRemRangeByScore(ctx, timedKey, "0", fmt.Sprintf("%d", cutoff))
	timedEntriesCmd := pipe.ZRange(ctx, timedKey, 0, -1)
	pipe.Expire(ctx, timedKey, window+time.Minute)
	if _, err := pipe.Exec(ctx); err != nil {
		return domain.FloodTrackResult{}, err
	}

	lastSender, err := s.client.Get(ctx, lastSenderKey).Result()
	if err != nil && err != goredis.Nil {
		return domain.FloodTrackResult{}, err
	}

	streakPipe := s.client.TxPipeline()
	if lastSender == fmt.Sprintf("%d", userID) {
		streakPipe.RPush(ctx, streakKey, messageID)
	} else {
		streakPipe.Del(ctx, streakKey)
		streakPipe.RPush(ctx, streakKey, messageID)
	}
	streakPipe.Set(ctx, lastSenderKey, fmt.Sprintf("%d", userID), streakTTL)
	streakPipe.Expire(ctx, streakKey, streakTTL)
	streakIDsCmd := streakPipe.LRange(ctx, streakKey, 0, -1)
	if _, err := streakPipe.Exec(ctx); err != nil {
		return domain.FloodTrackResult{}, err
	}

	result := domain.FloodTrackResult{
		ConsecutiveCount: int64(len(streakIDsCmd.Val())),
		TimedCount:       int64(len(timedEntriesCmd.Val())),
	}
	for _, raw := range streakIDsCmd.Val() {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			continue
		}
		result.ConsecutiveMessageIDs = append(result.ConsecutiveMessageIDs, id)
	}
	for _, raw := range timedEntriesCmd.Val() {
		parts := strings.SplitN(raw, ":", 2)
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			continue
		}
		result.TimedMessageIDs = append(result.TimedMessageIDs, id)
	}
	return result, nil
}

func (s *Store) ClearFlood(ctx context.Context, botID string, chatID int64, userID int64) error {
	timedKey := fmt.Sprintf("flood:timed:%s:%d:%d", botID, chatID, userID)
	streakKey := fmt.Sprintf("flood:streak:messages:%s:%d", botID, chatID)
	lastSenderKey := fmt.Sprintf("flood:streak:last:%s:%d", botID, chatID)

	lastSender, err := s.client.Get(ctx, lastSenderKey).Result()
	if err != nil && err != goredis.Nil {
		return err
	}

	keys := []string{timedKey}
	if lastSender == fmt.Sprintf("%d", userID) {
		keys = append(keys, streakKey, lastSenderKey)
	}
	return s.client.Del(ctx, keys...).Err()
}

func (s *Store) TrackJoinBurst(ctx context.Context, botID string, chatID int64, userID int64, window time.Duration) (int64, error) {
	key := fmt.Sprintf("joins:%s:%d", botID, chatID)
	now := time.Now()
	cutoff := now.Add(-window).UnixMilli()
	member := fmt.Sprintf("%d:%d", userID, now.UnixNano())

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

func (s *Store) AcquireLease(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return s.client.SetNX(ctx, "lease:"+key, "1", ttl).Result()
}

func (s *Store) SetLease(ctx context.Context, key string, ttl time.Duration) error {
	return s.client.Set(ctx, "lease:"+key, "1", ttl).Err()
}

func (s *Store) DeleteLease(ctx context.Context, key string) error {
	return s.client.Del(ctx, "lease:"+key).Err()
}
