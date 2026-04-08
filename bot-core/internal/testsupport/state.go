package testsupport

import (
	"context"
	"strconv"
	"sync"
	"time"

	"sukoon/bot-core/internal/domain"
)

type MemoryState struct {
	mu            sync.Mutex
	bots          map[string]domain.BotInstance
	flood         map[string][]timedFloodMessage
	streakUser    map[string]int64
	streakMessage map[string][]int64
	lease         map[string]time.Time
}

type timedFloodMessage struct {
	MessageID int64
	SeenAt    time.Time
}

func NewMemoryState() *MemoryState {
	return &MemoryState{
		bots:          map[string]domain.BotInstance{},
		flood:         map[string][]timedFloodMessage{},
		streakUser:    map[string]int64{},
		streakMessage: map[string][]int64{},
		lease:         map[string]time.Time{},
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

func (m *MemoryState) TrackFlood(_ context.Context, botID string, chatID int64, userID int64, messageID int64, window time.Duration) (domain.FloodTrackResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := botID + ":" + strconv.FormatInt(chatID, 10) + ":" + strconv.FormatInt(userID, 10)
	chatKey := botID + ":" + strconv.FormatInt(chatID, 10)
	now := time.Now()

	var kept []timedFloodMessage
	for _, ts := range m.flood[key] {
		if now.Sub(ts.SeenAt) <= window {
			kept = append(kept, ts)
		}
	}
	kept = append(kept, timedFloodMessage{MessageID: messageID, SeenAt: now})
	m.flood[key] = kept

	if m.streakUser[chatKey] == userID {
		m.streakMessage[chatKey] = append(m.streakMessage[chatKey], messageID)
	} else {
		m.streakUser[chatKey] = userID
		m.streakMessage[chatKey] = []int64{messageID}
	}

	result := domain.FloodTrackResult{
		ConsecutiveCount:      int64(len(m.streakMessage[chatKey])),
		ConsecutiveMessageIDs: append([]int64{}, m.streakMessage[chatKey]...),
		TimedCount:            int64(len(kept)),
	}
	for _, item := range kept {
		result.TimedMessageIDs = append(result.TimedMessageIDs, item.MessageID)
	}
	return result, nil
}

func (m *MemoryState) ClearFlood(_ context.Context, botID string, chatID int64, userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := botID + ":" + strconv.FormatInt(chatID, 10) + ":" + strconv.FormatInt(userID, 10)
	chatKey := botID + ":" + strconv.FormatInt(chatID, 10)
	delete(m.flood, key)
	if m.streakUser[chatKey] == userID {
		delete(m.streakUser, chatKey)
		delete(m.streakMessage, chatKey)
	}
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

func (m *MemoryState) SetLease(_ context.Context, key string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lease[key] = time.Now().Add(ttl)
	return nil
}

func (m *MemoryState) DeleteLease(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.lease, key)
	return nil
}
