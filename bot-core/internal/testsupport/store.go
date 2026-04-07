package testsupport

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/telegram"
)

type MemoryStore struct {
	mu               sync.Mutex
	botsByID         map[string]domain.BotInstance
	botsByWebhookKey map[string]string
	updateStates     map[int64]*updateState
	nextUpdateID     int64
	chats            map[string]telegram.Chat
	users            map[int64]domain.UserProfile
	settings         map[string]domain.ChatSettings
	moderation       map[string]domain.ModerationSettings
	antiflood        map[string]domain.AntifloodSettings
	captchaSettings  map[string]domain.CaptchaSettings
	antiabuse        map[string]domain.AntiAbuseSettings
	antibio          map[string]domain.AntiBioSettings
	roles            map[string]map[int64][]string
	chatRoles        map[string]map[int64][]string
	approvals        map[string]map[int64]bool
	disabled         map[string]map[string]struct{}
	warnings         map[string]int
	notes            map[string]map[string]domain.Note
	filters          map[string]map[string]domain.FilterRule
	locks            map[string]map[string]domain.LockRule
	blocklist        map[string][]domain.BlocklistRule
	antibioExempt    map[string]map[int64]bool
	globalUsers      map[string]domain.GlobalBlacklistUser
	globalChats      map[string]domain.GlobalBlacklistChat
	federations      map[string]domain.Federation
	federationChats  map[string][]string
	federationAdmins map[string]map[int64]domain.FederationAdmin
	federationBans   map[string]map[int64]domain.FederationBan
	nextBlocklistID  int64
	nextFilterID     int64
	challenges       map[string]domain.CaptchaChallenge
	afk              map[string]domain.AFKState
	jobs             map[string]domain.Job
}

type updateState struct {
	job         domain.QueuedUpdate
	status      string
	availableAt time.Time
	lastError   string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		botsByID:         map[string]domain.BotInstance{},
		botsByWebhookKey: map[string]string{},
		updateStates:     map[int64]*updateState{},
		chats:            map[string]telegram.Chat{},
		users:            map[int64]domain.UserProfile{},
		settings:         map[string]domain.ChatSettings{},
		moderation:       map[string]domain.ModerationSettings{},
		antiflood:        map[string]domain.AntifloodSettings{},
		captchaSettings:  map[string]domain.CaptchaSettings{},
		antiabuse:        map[string]domain.AntiAbuseSettings{},
		antibio:          map[string]domain.AntiBioSettings{},
		roles:            map[string]map[int64][]string{},
		chatRoles:        map[string]map[int64][]string{},
		approvals:        map[string]map[int64]bool{},
		disabled:         map[string]map[string]struct{}{},
		warnings:         map[string]int{},
		notes:            map[string]map[string]domain.Note{},
		filters:          map[string]map[string]domain.FilterRule{},
		locks:            map[string]map[string]domain.LockRule{},
		blocklist:        map[string][]domain.BlocklistRule{},
		antibioExempt:    map[string]map[int64]bool{},
		globalUsers:      map[string]domain.GlobalBlacklistUser{},
		globalChats:      map[string]domain.GlobalBlacklistChat{},
		federations:      map[string]domain.Federation{},
		federationChats:  map[string][]string{},
		federationAdmins: map[string]map[int64]domain.FederationAdmin{},
		federationBans:   map[string]map[int64]domain.FederationBan{},
		challenges:       map[string]domain.CaptchaChallenge{},
		afk:              map[string]domain.AFKState{},
		jobs:             map[string]domain.Job{},
	}
}

func (m *MemoryStore) QueuedUpdateCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.updateStates)
}

func (m *MemoryStore) PendingCaptchaForUser(botID string, chatID int64, userID int64) (domain.CaptchaChallenge, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, challenge := range m.challenges {
		if challenge.BotID == botID && challenge.ChatID == chatID && challenge.UserID == userID && challenge.Status == "pending" {
			return challenge, true
		}
	}
	return domain.CaptchaChallenge{}, false
}

func (m *MemoryStore) Close()                        {}
func (m *MemoryStore) Migrate(context.Context) error { return nil }

func (m *MemoryStore) UpsertPrimaryBot(_ context.Context, bot domain.BotInstance, ownerUserIDs []int64) (domain.BotInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	bot.Status = "active"
	m.botsByID[bot.ID] = bot
	m.botsByWebhookKey[bot.WebhookKey] = bot.ID
	if _, ok := m.roles[bot.ID]; !ok {
		m.roles[bot.ID] = map[int64][]string{}
	}
	for _, ownerID := range ownerUserIDs {
		m.roles[bot.ID][ownerID] = appendRole(m.roles[bot.ID][ownerID], "owner")
	}
	return bot, nil
}

func (m *MemoryStore) CreateCloneBot(_ context.Context, bot domain.BotInstance, ownerUserID int64) (domain.BotInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	bot.Status = "active"
	bot.CreatedByUserID = ownerUserID
	m.botsByID[bot.ID] = bot
	m.botsByWebhookKey[bot.WebhookKey] = bot.ID
	if _, ok := m.roles[bot.ID]; !ok {
		m.roles[bot.ID] = map[int64][]string{}
	}
	m.roles[bot.ID][ownerUserID] = appendRole(m.roles[bot.ID][ownerUserID], "owner")
	return bot, nil
}

func (m *MemoryStore) DeleteBotInstance(_ context.Context, botID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	bot, ok := m.botsByID[botID]
	if ok {
		delete(m.botsByWebhookKey, bot.WebhookKey)
	}
	delete(m.botsByID, botID)
	delete(m.roles, botID)
	return nil
}

func (m *MemoryStore) ResolveBotByWebhookKey(_ context.Context, webhookKey string) (domain.BotInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	botID, ok := m.botsByWebhookKey[webhookKey]
	if !ok {
		return domain.BotInstance{}, errors.New("bot not found")
	}
	return m.botsByID[botID], nil
}

func (m *MemoryStore) GetBotByID(_ context.Context, botID string) (domain.BotInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	bot, ok := m.botsByID[botID]
	if !ok {
		return domain.BotInstance{}, errors.New("bot not found")
	}
	return bot, nil
}

func (m *MemoryStore) ListOwnedBots(_ context.Context, ownerUserID int64) ([]domain.BotInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var bots []domain.BotInstance
	for botID, bot := range m.botsByID {
		for _, role := range m.roles[botID][ownerUserID] {
			if role == "owner" {
				bots = append(bots, bot)
				break
			}
		}
	}
	sort.Slice(bots, func(i, j int) bool { return bots[i].ID < bots[j].ID })
	return bots, nil
}

func (m *MemoryStore) EnqueueUpdate(_ context.Context, botID string, updateID int64, payload []byte) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, state := range m.updateStates {
		if state.job.BotID == botID && state.job.UpdateID == updateID {
			return false, nil
		}
	}
	m.nextUpdateID++
	m.updateStates[m.nextUpdateID] = &updateState{
		job: domain.QueuedUpdate{
			ID:       m.nextUpdateID,
			BotID:    botID,
			UpdateID: updateID,
			Payload:  payload,
		},
		status:      "pending",
		availableAt: time.Now(),
	}
	return true, nil
}

func (m *MemoryStore) ClaimPendingUpdates(_ context.Context, workerID string, limit int) ([]domain.QueuedUpdate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	_ = workerID
	now := time.Now()
	var ids []int64
	for id, state := range m.updateStates {
		if state.status == "pending" && !state.availableAt.After(now) {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(ids) > limit {
		ids = ids[:limit]
	}

	updates := make([]domain.QueuedUpdate, 0, len(ids))
	for _, id := range ids {
		state := m.updateStates[id]
		state.status = "processing"
		state.job.Attempts++
		updates = append(updates, state.job)
	}
	return updates, nil
}

func (m *MemoryStore) MarkUpdateCompleted(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if state, ok := m.updateStates[id]; ok {
		state.status = "completed"
	}
	return nil
}

func (m *MemoryStore) MarkUpdateRetry(_ context.Context, id int64, attempts int, lastError string, availableAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if state, ok := m.updateStates[id]; ok {
		state.status = "pending"
		state.job.Attempts = attempts
		state.lastError = lastError
		state.availableAt = availableAt
	}
	return nil
}

func (m *MemoryStore) MarkUpdateDead(_ context.Context, id int64, lastError string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if state, ok := m.updateStates[id]; ok {
		state.status = "dead"
		state.lastError = lastError
	}
	return nil
}

func (m *MemoryStore) EnsureChat(_ context.Context, botID string, chat telegram.Chat) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(botID, chat.ID)
	m.chats[key] = chat
	if _, ok := m.settings[key]; !ok {
		m.settings[key] = domain.ChatSettings{BotID: botID, ChatID: chat.ID, Language: "en"}
	}
	if _, ok := m.moderation[key]; !ok {
		m.moderation[key] = domain.ModerationSettings{BotID: botID, ChatID: chat.ID, WarnLimit: 3, WarnMode: "mute"}
	}
	if _, ok := m.antiflood[key]; !ok {
		m.antiflood[key] = domain.AntifloodSettings{BotID: botID, ChatID: chat.ID, Limit: 6, WindowSeconds: 10, Action: "mute"}
	}
	if _, ok := m.captchaSettings[key]; !ok {
		m.captchaSettings[key] = domain.CaptchaSettings{BotID: botID, ChatID: chat.ID, Mode: "button", TimeoutSeconds: 120, FailureAction: "kick", ChallengeDigits: 2}
	}
	if _, ok := m.antiabuse[key]; !ok {
		m.antiabuse[key] = domain.AntiAbuseSettings{BotID: botID, ChatID: chat.ID, Enabled: false, Action: "delete_warn"}
	}
	if _, ok := m.antibio[key]; !ok {
		m.antibio[key] = domain.AntiBioSettings{BotID: botID, ChatID: chat.ID, Enabled: false, Action: "kick"}
	}
	return nil
}

func (m *MemoryStore) EnsureUser(_ context.Context, user telegram.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.ID] = domain.UserProfile{
		ID:        user.ID,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		IsBot:     user.IsBot,
	}
	return nil
}

func (m *MemoryStore) GetUserByID(_ context.Context, userID int64) (domain.UserProfile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.users[userID]
	if !ok {
		return domain.UserProfile{}, pgx.ErrNoRows
	}
	return user, nil
}

func (m *MemoryStore) GetUserByUsername(_ context.Context, username string) (domain.UserProfile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	username = strings.TrimPrefix(strings.ToLower(username), "@")
	for _, user := range m.users {
		if strings.ToLower(user.Username) == username {
			return user, nil
		}
	}
	return domain.UserProfile{}, pgx.ErrNoRows
}

func (m *MemoryStore) LoadRuntimeBundle(_ context.Context, botID string, chatID int64) (domain.RuntimeBundle, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(botID, chatID)
	bundle := domain.RuntimeBundle{
		Settings:         m.settings[key],
		Moderation:       m.moderation[key],
		Antiflood:        m.antiflood[key],
		Captcha:          m.captchaSettings[key],
		AntiAbuse:        m.antiabuse[key],
		AntiBio:          m.antibio[key],
		DisabledCommands: map[string]struct{}{},
		Locks:            map[string]domain.LockRule{},
	}
	for command := range m.disabled[key] {
		bundle.DisabledCommands[command] = struct{}{}
	}
	for name, lock := range m.locks[key] {
		bundle.Locks[name] = lock
	}
	bundle.Blocklist = append(bundle.Blocklist, m.blocklist[key]...)
	return bundle, nil
}

func (m *MemoryStore) GetBotRoles(_ context.Context, botID string, userID int64) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.roles[botID][userID]...), nil
}

func (m *MemoryStore) IsApproved(_ context.Context, botID string, chatID int64, userID int64) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.approvals[chatKey(botID, chatID)][userID], nil
}

func (m *MemoryStore) SetApproval(_ context.Context, botID string, chatID int64, userID int64, approvedBy int64, approved bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_ = approvedBy
	key := chatKey(botID, chatID)
	if _, ok := m.approvals[key]; !ok {
		m.approvals[key] = map[int64]bool{}
	}
	if approved {
		m.approvals[key][userID] = true
	} else {
		delete(m.approvals[key], userID)
	}
	return nil
}

func (m *MemoryStore) ListApprovedUsers(_ context.Context, botID string, chatID int64) ([]int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var users []int64
	for userID := range m.approvals[chatKey(botID, chatID)] {
		users = append(users, userID)
	}
	sort.Slice(users, func(i, j int) bool { return users[i] < users[j] })
	return users, nil
}

func (m *MemoryStore) SetDisabledCommand(_ context.Context, botID string, chatID int64, command string, disabled bool, changedBy int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_ = changedBy
	key := chatKey(botID, chatID)
	if _, ok := m.disabled[key]; !ok {
		m.disabled[key] = map[string]struct{}{}
	}
	if disabled {
		m.disabled[key][command] = struct{}{}
	} else {
		delete(m.disabled[key], command)
	}
	return nil
}

func (m *MemoryStore) SetWarnConfig(_ context.Context, botID string, chatID int64, limit int, mode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(botID, chatID)
	current := m.moderation[key]
	current.WarnLimit = limit
	current.WarnMode = mode
	m.moderation[key] = current
	return nil
}

func (m *MemoryStore) IncrementWarnings(_ context.Context, botID string, chatID int64, userID int64, reason string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_ = reason
	key := userKey(botID, chatID, userID)
	m.warnings[key]++
	return m.warnings[key], nil
}

func (m *MemoryStore) ResetWarnings(_ context.Context, botID string, chatID int64, userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.warnings, userKey(botID, chatID, userID))
	return nil
}

func (m *MemoryStore) GetWarnings(_ context.Context, botID string, chatID int64, userID int64) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.warnings[userKey(botID, chatID, userID)], nil
}

func (m *MemoryStore) SetLogChannel(_ context.Context, botID string, chatID int64, logChannelID *int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(botID, chatID)
	settings := m.settings[key]
	settings.LogChannelID = logChannelID
	m.settings[key] = settings
	return nil
}

func (m *MemoryStore) SetReportsEnabled(_ context.Context, botID string, chatID int64, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(botID, chatID)
	settings := m.settings[key]
	settings.ReportsEnabled = enabled
	m.settings[key] = settings
	return nil
}

func (m *MemoryStore) SetCleanCommands(_ context.Context, botID string, chatID int64, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(botID, chatID)
	settings := m.settings[key]
	settings.CleanCommands = enabled
	m.settings[key] = settings
	return nil
}

func (m *MemoryStore) SetCleanService(_ context.Context, botID string, chatID int64, target string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(botID, chatID)
	settings := m.settings[key]
	switch target {
	case "join":
		settings.CleanServiceJoin = enabled
	case "leave":
		settings.CleanServiceLeave = enabled
	case "pin":
		settings.CleanServicePin = enabled
	case "title":
		settings.CleanServiceTitle = enabled
	case "photo":
		settings.CleanServicePhoto = enabled
	case "other":
		settings.CleanServiceOther = enabled
	case "videochat":
		settings.CleanServiceVideoChat = enabled
	default:
		settings.CleanServiceJoin = enabled
		settings.CleanServiceLeave = enabled
		settings.CleanServicePin = enabled
		settings.CleanServiceTitle = enabled
		settings.CleanServicePhoto = enabled
		settings.CleanServiceOther = enabled
		settings.CleanServiceVideoChat = enabled
	}
	m.settings[key] = settings
	return nil
}

func (m *MemoryStore) UpsertNote(_ context.Context, note domain.Note) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(note.BotID, note.ChatID)
	if _, ok := m.notes[key]; !ok {
		m.notes[key] = map[string]domain.Note{}
	}
	m.notes[key][strings.ToLower(note.Name)] = note
	return nil
}

func (m *MemoryStore) GetNote(_ context.Context, botID string, chatID int64, name string) (domain.Note, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	note, ok := m.notes[chatKey(botID, chatID)][strings.ToLower(name)]
	if !ok {
		return domain.Note{}, pgx.ErrNoRows
	}
	return note, nil
}

func (m *MemoryStore) ListNotes(_ context.Context, botID string, chatID int64) ([]domain.Note, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(botID, chatID)
	var notes []domain.Note
	for _, note := range m.notes[key] {
		notes = append(notes, note)
	}
	sort.Slice(notes, func(i, j int) bool { return notes[i].Name < notes[j].Name })
	return notes, nil
}

func (m *MemoryStore) DeleteNote(_ context.Context, botID string, chatID int64, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.notes[chatKey(botID, chatID)], strings.ToLower(name))
	return nil
}

func (m *MemoryStore) UpsertFilter(_ context.Context, filter domain.FilterRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(filter.BotID, filter.ChatID)
	if _, ok := m.filters[key]; !ok {
		m.filters[key] = map[string]domain.FilterRule{}
	}
	m.nextFilterID++
	filter.ID = m.nextFilterID
	m.filters[key][strings.ToLower(filter.Trigger)] = filter
	return nil
}

func (m *MemoryStore) DeleteFilter(_ context.Context, botID string, chatID int64, trigger string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.filters[chatKey(botID, chatID)], strings.ToLower(trigger))
	return nil
}

func (m *MemoryStore) ListFilters(_ context.Context, botID string, chatID int64) ([]domain.FilterRule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(botID, chatID)
	var rules []domain.FilterRule
	for _, rule := range m.filters[key] {
		rules = append(rules, rule)
	}
	sort.Slice(rules, func(i, j int) bool { return rules[i].ID < rules[j].ID })
	return rules, nil
}

func (m *MemoryStore) SetWelcome(_ context.Context, botID string, chatID int64, enabled bool, text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(botID, chatID)
	settings := m.settings[key]
	settings.WelcomeEnabled = enabled
	settings.WelcomeText = text
	m.settings[key] = settings
	return nil
}

func (m *MemoryStore) SetGoodbye(_ context.Context, botID string, chatID int64, enabled bool, text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(botID, chatID)
	settings := m.settings[key]
	settings.GoodbyeEnabled = enabled
	settings.GoodbyeText = text
	m.settings[key] = settings
	return nil
}

func (m *MemoryStore) SetRules(_ context.Context, botID string, chatID int64, text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(botID, chatID)
	settings := m.settings[key]
	settings.RulesText = text
	m.settings[key] = settings
	return nil
}

func (m *MemoryStore) UpsertLock(_ context.Context, lock domain.LockRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(lock.BotID, lock.ChatID)
	if _, ok := m.locks[key]; !ok {
		m.locks[key] = map[string]domain.LockRule{}
	}
	m.locks[key][lock.LockType] = lock
	return nil
}

func (m *MemoryStore) DeleteLock(_ context.Context, botID string, chatID int64, lockType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.locks[chatKey(botID, chatID)], lockType)
	return nil
}

func (m *MemoryStore) ListLocks(_ context.Context, botID string, chatID int64) ([]domain.LockRule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var locks []domain.LockRule
	for _, lock := range m.locks[chatKey(botID, chatID)] {
		locks = append(locks, lock)
	}
	return locks, nil
}

func (m *MemoryStore) AddBlocklistRule(_ context.Context, rule domain.BlocklistRule) (domain.BlocklistRule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(rule.BotID, rule.ChatID)
	m.nextBlocklistID++
	rule.ID = m.nextBlocklistID
	m.blocklist[key] = append(m.blocklist[key], rule)
	return rule, nil
}

func (m *MemoryStore) DeleteBlocklistRule(_ context.Context, botID string, chatID int64, pattern string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(botID, chatID)
	var filtered []domain.BlocklistRule
	for _, rule := range m.blocklist[key] {
		if rule.Pattern != pattern {
			filtered = append(filtered, rule)
		}
	}
	m.blocklist[key] = filtered
	return nil
}

func (m *MemoryStore) ListBlocklistRules(_ context.Context, botID string, chatID int64) ([]domain.BlocklistRule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]domain.BlocklistRule{}, m.blocklist[chatKey(botID, chatID)]...), nil
}

func (m *MemoryStore) SetAntiflood(_ context.Context, settings domain.AntifloodSettings) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.antiflood[chatKey(settings.BotID, settings.ChatID)] = settings
	return nil
}

func (m *MemoryStore) SetCaptchaSettings(_ context.Context, settings domain.CaptchaSettings) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.captchaSettings[chatKey(settings.BotID, settings.ChatID)] = settings
	return nil
}

func (m *MemoryStore) CreateCaptchaChallenge(_ context.Context, challenge domain.CaptchaChallenge) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.challenges[challenge.ID] = challenge
	return nil
}

func (m *MemoryStore) GetPendingCaptchaChallenge(_ context.Context, botID string, chatID int64, userID int64) (domain.CaptchaChallenge, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, challenge := range m.challenges {
		if challenge.BotID == botID && challenge.ChatID == chatID && challenge.UserID == userID && challenge.Status == "pending" {
			return challenge, nil
		}
	}
	return domain.CaptchaChallenge{}, errors.New("captcha challenge not found")
}

func (m *MemoryStore) MarkCaptchaSolved(_ context.Context, challengeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	challenge := m.challenges[challengeID]
	challenge.Status = "solved"
	m.challenges[challengeID] = challenge
	return nil
}

func (m *MemoryStore) ListExpiredCaptchaChallenges(_ context.Context, now time.Time, limit int) ([]domain.CaptchaChallenge, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.CaptchaChallenge
	for _, challenge := range m.challenges {
		if challenge.Status == "pending" && !challenge.ExpiresAt.After(now) {
			result = append(result, challenge)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ExpiresAt.Before(result[j].ExpiresAt) })
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *MemoryStore) MarkCaptchaExpired(_ context.Context, challengeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	challenge := m.challenges[challengeID]
	challenge.Status = "expired"
	m.challenges[challengeID] = challenge
	return nil
}

func (m *MemoryStore) SetAFK(_ context.Context, state domain.AFKState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.afk[afkKey(state.BotID, state.UserID)] = state
	return nil
}

func (m *MemoryStore) ClearAFK(_ context.Context, botID string, userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.afk, afkKey(botID, userID))
	return nil
}

func (m *MemoryStore) GetAFK(_ context.Context, botID string, userID int64) (domain.AFKState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.afk[afkKey(botID, userID)], nil
}

func (m *MemoryStore) CreateJob(_ context.Context, job domain.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[job.ID] = job
	return nil
}

func (m *MemoryStore) UpdateJobProgress(_ context.Context, jobID string, status string, progress int, total int, errText string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job := m.jobs[jobID]
	job.Status = status
	job.Progress = progress
	job.Total = total
	job.Error = errText
	m.jobs[jobID] = job
	return nil
}

func appendRole(existing []string, role string) []string {
	for _, current := range existing {
		if current == role {
			return existing
		}
	}
	return append(existing, role)
}

func chatKey(botID string, chatID int64) string {
	return botID + ":" + strconv.FormatInt(chatID, 10)
}

func userKey(botID string, chatID int64, userID int64) string {
	return chatKey(botID, chatID) + ":" + strconv.FormatInt(userID, 10)
}

func afkKey(botID string, userID int64) string {
	return botID + ":" + strconv.FormatInt(userID, 10)
}
