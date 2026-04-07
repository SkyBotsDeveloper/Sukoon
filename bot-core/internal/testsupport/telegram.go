package testsupport

import (
	"context"
	"fmt"
	"sync"
	"time"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/telegram"
)

type FakeTelegramClient struct {
	mu               sync.Mutex
	Messages         []SentMessage
	DeletedMessages  []DeletedMessage
	PinnedMessages   []PinnedMessage
	UnpinnedMessages []UnpinnedMessage
	UnpinAllChats    []int64
	Bans             []BanCall
	Unbans           []UnbanCall
	Restrictions     []RestrictionCall
	LeftChats        []int64
	Webhooks         []telegram.SetWebhookOptions
	DeletedWebhooks  int
	CallbackAnswers  []CallbackAnswer
	AdminsByChat     map[int64][]telegram.ChatAdministrator
	ChatsByID        map[int64]telegram.Chat
	Me               telegram.User
	SendErrors       map[int64]error
	DeleteErrors     map[int64]error
	BanErrors        map[string]error
	UnbanErrors      map[string]error
	nextMessageID    int64
}

type StaticClientFactory struct {
	Client *FakeTelegramClient
}

type SentMessage struct {
	MessageID int64
	ChatID    int64
	Text      string
	Options   telegram.SendMessageOptions
}

type DeletedMessage struct {
	ChatID    int64
	MessageID int64
}

type PinnedMessage struct {
	ChatID              int64
	MessageID           int64
	DisableNotification bool
}

type UnpinnedMessage struct {
	ChatID    int64
	MessageID *int64
}

type BanCall struct {
	ChatID int64
	UserID int64
	Until  *time.Time
}

type UnbanCall struct {
	ChatID int64
	UserID int64
}

type RestrictionCall struct {
	ChatID      int64
	UserID      int64
	Permissions telegram.RestrictPermissions
	Until       *time.Time
}

type CallbackAnswer struct {
	ID        string
	Text      string
	ShowAlert bool
}

func NewFakeTelegramClient() *FakeTelegramClient {
	return &FakeTelegramClient{
		AdminsByChat:  map[int64][]telegram.ChatAdministrator{},
		ChatsByID:     map[int64]telegram.Chat{},
		SendErrors:    map[int64]error{},
		DeleteErrors:  map[int64]error{},
		BanErrors:     map[string]error{},
		UnbanErrors:   map[string]error{},
		nextMessageID: 100,
	}
}

func (f StaticClientFactory) ForBot(bot domain.BotInstance) telegram.Client {
	_ = bot
	return f.Client
}

func (f *FakeTelegramClient) SendMessage(_ context.Context, chatID int64, text string, options telegram.SendMessageOptions) (telegram.Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err, ok := f.SendErrors[chatID]; ok {
		return telegram.Message{}, err
	}
	f.nextMessageID++
	f.Messages = append(f.Messages, SentMessage{MessageID: f.nextMessageID, ChatID: chatID, Text: text, Options: options})
	return telegram.Message{
		MessageID: f.nextMessageID,
		Chat:      telegram.Chat{ID: chatID},
		Text:      text,
	}, nil
}

func (f *FakeTelegramClient) DeleteMessage(_ context.Context, chatID int64, messageID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err, ok := f.DeleteErrors[messageID]; ok {
		return err
	}
	f.DeletedMessages = append(f.DeletedMessages, DeletedMessage{ChatID: chatID, MessageID: messageID})
	return nil
}

func (f *FakeTelegramClient) PinChatMessage(_ context.Context, chatID int64, messageID int64, disableNotification bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.PinnedMessages = append(f.PinnedMessages, PinnedMessage{ChatID: chatID, MessageID: messageID, DisableNotification: disableNotification})
	return nil
}

func (f *FakeTelegramClient) UnpinChatMessage(_ context.Context, chatID int64, messageID *int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.UnpinnedMessages = append(f.UnpinnedMessages, UnpinnedMessage{ChatID: chatID, MessageID: messageID})
	return nil
}

func (f *FakeTelegramClient) UnpinAllChatMessages(_ context.Context, chatID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.UnpinAllChats = append(f.UnpinAllChats, chatID)
	return nil
}

func (f *FakeTelegramClient) BanChatMember(_ context.Context, chatID int64, userID int64, untilDate *time.Time, revokeMessages bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	_ = revokeMessages
	if err, ok := f.BanErrors[banKey(chatID, userID)]; ok {
		return err
	}
	f.Bans = append(f.Bans, BanCall{ChatID: chatID, UserID: userID, Until: untilDate})
	return nil
}

func (f *FakeTelegramClient) UnbanChatMember(_ context.Context, chatID int64, userID int64, onlyIfBanned bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	_ = onlyIfBanned
	if err, ok := f.UnbanErrors[banKey(chatID, userID)]; ok {
		return err
	}
	f.Unbans = append(f.Unbans, UnbanCall{ChatID: chatID, UserID: userID})
	return nil
}

func (f *FakeTelegramClient) RestrictChatMember(_ context.Context, chatID int64, userID int64, permissions telegram.RestrictPermissions, untilDate *time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Restrictions = append(f.Restrictions, RestrictionCall{ChatID: chatID, UserID: userID, Permissions: permissions, Until: untilDate})
	return nil
}

func (f *FakeTelegramClient) GetChatAdministrators(_ context.Context, chatID int64) ([]telegram.ChatAdministrator, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]telegram.ChatAdministrator{}, f.AdminsByChat[chatID]...), nil
}

func (f *FakeTelegramClient) GetChat(_ context.Context, chatID int64) (telegram.Chat, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.ChatsByID[chatID], nil
}

func (f *FakeTelegramClient) GetMe(_ context.Context) (telegram.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Me, nil
}

func (f *FakeTelegramClient) SetWebhook(_ context.Context, options telegram.SetWebhookOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Webhooks = append(f.Webhooks, options)
	return nil
}

func (f *FakeTelegramClient) DeleteWebhook(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.DeletedWebhooks++
	return nil
}

func (f *FakeTelegramClient) LeaveChat(_ context.Context, chatID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.LeftChats = append(f.LeftChats, chatID)
	return nil
}

func (f *FakeTelegramClient) AnswerCallbackQuery(_ context.Context, callbackQueryID string, text string, showAlert bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.CallbackAnswers = append(f.CallbackAnswers, CallbackAnswer{ID: callbackQueryID, Text: text, ShowAlert: showAlert})
	return nil
}

func banKey(chatID int64, userID int64) string {
	return fmt.Sprintf("%d:%d", chatID, userID)
}
