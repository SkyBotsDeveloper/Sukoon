package persistence

import (
	"context"
	"time"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/telegram"
)

type Store interface {
	Close()
	Migrate(ctx context.Context) error
	UpsertPrimaryBot(ctx context.Context, bot domain.BotInstance, ownerUserIDs []int64) (domain.BotInstance, error)
	CreateCloneBot(ctx context.Context, bot domain.BotInstance, ownerUserID int64) (domain.BotInstance, error)
	DeleteBotInstance(ctx context.Context, botID string) error
	ResolveBotByWebhookKey(ctx context.Context, webhookKey string) (domain.BotInstance, error)
	GetBotByID(ctx context.Context, botID string) (domain.BotInstance, error)
	ListOwnedBots(ctx context.Context, ownerUserID int64) ([]domain.BotInstance, error)
	EnqueueUpdate(ctx context.Context, botID string, updateID int64, payload []byte) (bool, error)
	ClaimPendingUpdates(ctx context.Context, workerID string, limit int) ([]domain.QueuedUpdate, error)
	MarkUpdateCompleted(ctx context.Context, id int64) error
	MarkUpdateRetry(ctx context.Context, id int64, attempts int, lastError string, availableAt time.Time) error
	MarkUpdateDead(ctx context.Context, id int64, lastError string) error
	EnsureChat(ctx context.Context, botID string, chat telegram.Chat) error
	EnsureUser(ctx context.Context, user telegram.User) error
	GetUserByID(ctx context.Context, userID int64) (domain.UserProfile, error)
	GetUserByUsername(ctx context.Context, username string) (domain.UserProfile, error)
	LoadRuntimeBundle(ctx context.Context, botID string, chatID int64) (domain.RuntimeBundle, error)
	GetBotRoles(ctx context.Context, botID string, userID int64) ([]string, error)
	SetBotRole(ctx context.Context, botID string, userID int64, role string, enabled bool) error
	ListBotRoleUsers(ctx context.Context, botID string, role string) ([]domain.UserProfile, error)
	GetChatRoles(ctx context.Context, botID string, chatID int64, userID int64) ([]string, error)
	SetChatRole(ctx context.Context, botID string, chatID int64, userID int64, role string, grantedBy int64, enabled bool) error
	ListChatRoleUsers(ctx context.Context, botID string, chatID int64, role string) ([]domain.UserProfile, error)
	IsApproved(ctx context.Context, botID string, chatID int64, userID int64) (bool, error)
	SetApproval(ctx context.Context, botID string, chatID int64, userID int64, approvedBy int64, approved bool) error
	ListApprovedUsers(ctx context.Context, botID string, chatID int64) ([]int64, error)
	SetDisabledCommand(ctx context.Context, botID string, chatID int64, command string, disabled bool, changedBy int64) error
	SetDisabledDelete(ctx context.Context, botID string, chatID int64, enabled bool) error
	SetDisableAdmins(ctx context.Context, botID string, chatID int64, enabled bool) error
	SetAdminErrors(ctx context.Context, botID string, chatID int64, enabled bool) error
	SetAnonAdmins(ctx context.Context, botID string, chatID int64, enabled bool) error
	SetWarnConfig(ctx context.Context, botID string, chatID int64, limit int, mode string) error
	IncrementWarnings(ctx context.Context, botID string, chatID int64, userID int64, reason string) (int, error)
	ResetWarnings(ctx context.Context, botID string, chatID int64, userID int64) error
	GetWarnings(ctx context.Context, botID string, chatID int64, userID int64) (int, error)
	SetLogChannel(ctx context.Context, botID string, chatID int64, logChannelID *int64) error
	SetReportsEnabled(ctx context.Context, botID string, chatID int64, enabled bool) error
	SetCleanCommands(ctx context.Context, botID string, chatID int64, enabled bool) error
	SetCleanService(ctx context.Context, botID string, chatID int64, target string, enabled bool) error
	SetLanguage(ctx context.Context, botID string, chatID int64, language string) error
	UpsertNote(ctx context.Context, note domain.Note) error
	GetNote(ctx context.Context, botID string, chatID int64, name string) (domain.Note, error)
	ListNotes(ctx context.Context, botID string, chatID int64) ([]domain.Note, error)
	DeleteNote(ctx context.Context, botID string, chatID int64, name string) error
	UpsertFilter(ctx context.Context, filter domain.FilterRule) error
	DeleteFilter(ctx context.Context, botID string, chatID int64, trigger string) error
	ListFilters(ctx context.Context, botID string, chatID int64) ([]domain.FilterRule, error)
	SetWelcome(ctx context.Context, botID string, chatID int64, enabled bool, text string) error
	SetGoodbye(ctx context.Context, botID string, chatID int64, enabled bool, text string) error
	SetRules(ctx context.Context, botID string, chatID int64, text string) error
	UpsertLock(ctx context.Context, lock domain.LockRule) error
	DeleteLock(ctx context.Context, botID string, chatID int64, lockType string) error
	ListLocks(ctx context.Context, botID string, chatID int64) ([]domain.LockRule, error)
	AddBlocklistRule(ctx context.Context, rule domain.BlocklistRule) (domain.BlocklistRule, error)
	DeleteBlocklistRule(ctx context.Context, botID string, chatID int64, pattern string) error
	ListBlocklistRules(ctx context.Context, botID string, chatID int64) ([]domain.BlocklistRule, error)
	SetAntiflood(ctx context.Context, settings domain.AntifloodSettings) error
	SetAntiAbuseSettings(ctx context.Context, settings domain.AntiAbuseSettings) error
	SetAntiBioSettings(ctx context.Context, settings domain.AntiBioSettings) error
	IsAntiBioExempt(ctx context.Context, botID string, chatID int64, userID int64) (bool, error)
	SetAntiBioExemption(ctx context.Context, botID string, chatID int64, userID int64, addedBy int64, exempt bool) error
	ListAntiBioExemptions(ctx context.Context, botID string, chatID int64) ([]domain.UserProfile, error)
	SetCaptchaSettings(ctx context.Context, settings domain.CaptchaSettings) error
	CreateCaptchaChallenge(ctx context.Context, challenge domain.CaptchaChallenge) error
	GetPendingCaptchaChallenge(ctx context.Context, botID string, chatID int64, userID int64) (domain.CaptchaChallenge, error)
	MarkCaptchaSolved(ctx context.Context, challengeID string) error
	ListExpiredCaptchaChallenges(ctx context.Context, now time.Time, limit int) ([]domain.CaptchaChallenge, error)
	MarkCaptchaExpired(ctx context.Context, challengeID string) error
	SetAFK(ctx context.Context, state domain.AFKState) error
	ClearAFK(ctx context.Context, botID string, userID int64) error
	GetAFK(ctx context.Context, botID string, userID int64) (domain.AFKState, error)
	CreateJob(ctx context.Context, job domain.Job) error
	ClaimPendingJobs(ctx context.Context, workerID string, limit int) ([]domain.Job, error)
	GetJob(ctx context.Context, jobID string) (domain.Job, error)
	ListRecentJobs(ctx context.Context, botID string, limit int) ([]domain.Job, error)
	UpdateJobProgress(ctx context.Context, jobID string, status string, progress int, total int, errText string) error
	MarkJobCompleted(ctx context.Context, jobID string, resultSummary string) error
	MarkJobRetry(ctx context.Context, jobID string, attempts int, errText string, availableAt time.Time) error
	MarkJobDead(ctx context.Context, jobID string, errText string) error
	ListChats(ctx context.Context, botID string) ([]telegram.Chat, error)
	GetStats(ctx context.Context, botID string) (domain.BotStats, error)
	SetGlobalBlacklistUser(ctx context.Context, entry domain.GlobalBlacklistUser, enabled bool) error
	GetGlobalBlacklistUser(ctx context.Context, botID string, userID int64) (domain.GlobalBlacklistUser, bool, error)
	ListGlobalBlacklistUsers(ctx context.Context, botID string) ([]domain.GlobalBlacklistUser, error)
	SetGlobalBlacklistChat(ctx context.Context, entry domain.GlobalBlacklistChat, enabled bool) error
	GetGlobalBlacklistChat(ctx context.Context, botID string, chatID int64) (domain.GlobalBlacklistChat, bool, error)
	ListGlobalBlacklistChats(ctx context.Context, botID string) ([]domain.GlobalBlacklistChat, error)
	CreateFederation(ctx context.Context, federation domain.Federation) (domain.Federation, error)
	DeleteFederation(ctx context.Context, federationID string) error
	RenameFederation(ctx context.Context, federationID string, shortName string, displayName string) error
	GetFederationByID(ctx context.Context, federationID string) (domain.Federation, error)
	GetFederationByShortName(ctx context.Context, botID string, shortName string) (domain.Federation, error)
	GetFederationByChat(ctx context.Context, botID string, chatID int64) (domain.Federation, error)
	ListFederationsForUser(ctx context.Context, botID string, userID int64) ([]domain.Federation, error)
	JoinFederation(ctx context.Context, federationID string, botID string, chatID int64) error
	LeaveFederation(ctx context.Context, federationID string, botID string, chatID int64) error
	ListFederationChats(ctx context.Context, federationID string) ([]telegram.Chat, error)
	SetFederationAdmin(ctx context.Context, federationID string, userID int64, role string, enabled bool) error
	ListFederationAdmins(ctx context.Context, federationID string) ([]domain.FederationAdmin, error)
	TransferFederation(ctx context.Context, federationID string, newOwnerUserID int64) error
	SetFederationBan(ctx context.Context, ban domain.FederationBan, enabled bool) error
	GetFederationBan(ctx context.Context, federationID string, userID int64) (domain.FederationBan, bool, error)
	ExportUserData(ctx context.Context, botID string, userID int64) (map[string]any, error)
	DeleteUserData(ctx context.Context, botID string, userID int64) error
}
