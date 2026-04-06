package testsupport

import (
	"context"
	"strconv"
	"sync"
	"time"

	"sukoon/bot-core/internal/domain"
)

type MemoryState struct {
	mu    sync.Mutex
	bots  map[string]domain.BotInstance
	flood map[string][]time.Time
	lease map[string]time.Time
}

func NewMemoryState() *MemoryState {
	return &MemoryState{
		bots:  map[string]domain.BotInstance{},
		flood: map[string][]time.Time{},
		lease: map[string]time.Time{},
	}
}

func (m *MemoryState) Close() error { return nil }

func (m *MemoryState) GetCachedBot(_ context.Context, webhookKey string) (domain.BotInstance, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	bot, ok := m.bots[webhookKey]
	return bot, ok, nil
}

func (m *MemoryState) CacheBot(_ context.Context, bot domain.BotInstance, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_ = ttl
	m.bots[bot.WebhookKey] = bot
	return nil
}

func (m *MemoryState) TrackFlood(_ context.Context, botID string, chatID int64, userID int64, messageID int64, window time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_ = messageID
	key := botID + ":" + strconv.FormatInt(chatID, 10) + ":" + strconv.FormatInt(userID, 10)
	now := time.Now()
	var kept []time.Time
	for _, ts := range m.flood[key] {
		if now.Sub(ts) <= window {
			kept = append(kept, ts)
		}
	}
	kept = append(kept, now)
	m.flood[key] = kept
	return int64(len(kept)), nil
}

func (m *MemoryState) ClearFlood(_ context.Context, botID string, chatID int64, userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := botID + ":" + strconv.FormatInt(chatID, 10) + ":" + strconv.FormatInt(userID, 10)
	delete(m.flood, key)
	return nil
}

func (m *MemoryState) AcquireLease(_ context.Context, key string, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	if expiresAt, ok := m.lease[key]; ok && expiresAt.After(now) {
		return false, nil
	}
	m.lease[key] = now.Add(ttl)
	return true, nil
}
