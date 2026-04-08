package domain

import "time"

type BotInstance struct {
	ID              string
	Slug            string
	DisplayName     string
	TelegramToken   string
	WebhookKey      string
	WebhookSecret   string
	Username        string
	IsPrimary       bool
	CreatedByUserID int64
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type QueuedUpdate struct {
	ID        int64
	BotID     string
	UpdateID  int64
	Payload   []byte
	Attempts  int
	CreatedAt time.Time
}

type ChatSettings struct {
	BotID                 string
	ChatID                int64
	Language              string
	ReportsEnabled        bool
	LogChannelID          *int64
	CleanCommands         bool
	DisabledDelete        bool
	DisableAdmins         bool
	AdminErrors           bool
	AnonAdmins            bool
	CleanServiceJoin      bool
	CleanServiceLeave     bool
	CleanServicePin       bool
	CleanServiceTitle     bool
	CleanServicePhoto     bool
	CleanServiceOther     bool
	CleanServiceVideoChat bool
	WelcomeEnabled        bool
	WelcomeText           string
	GoodbyeEnabled        bool
	GoodbyeText           string
	RulesText             string
}

type ModerationSettings struct {
	BotID     string
	ChatID    int64
	WarnLimit int
	WarnMode  string
}

type AntifloodSettings struct {
	BotID         string
	ChatID        int64
	Enabled       bool
	Limit         int
	WindowSeconds int
	Action        string
}

type CaptchaSettings struct {
	BotID           string
	ChatID          int64
	Enabled         bool
	Mode            string
	TimeoutSeconds  int
	FailureAction   string
	ChallengeDigits int
}

type LockRule struct {
	BotID    string
	ChatID   int64
	LockType string
	Action   string
}

type BlocklistRule struct {
	ID        int64
	BotID     string
	ChatID    int64
	Pattern   string
	MatchMode string
	Action    string
	CreatedBy int64
	CreatedAt time.Time
}

type Note struct {
	BotID       string
	ChatID      int64
	Name        string
	Text        string
	ParseMode   string
	ButtonsJSON string
	CreatedBy   int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type FilterRule struct {
	ID           int64
	BotID        string
	ChatID       int64
	Trigger      string
	MatchMode    string
	ResponseText string
	ParseMode    string
	ButtonsJSON  string
	CreatedBy    int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CaptchaChallenge struct {
	ID            string
	BotID         string
	ChatID        int64
	UserID        int64
	Prompt        string
	Answer        string
	MessageID     int64
	ExpiresAt     time.Time
	Status        string
	FailureAction string
}

type AFKState struct {
	BotID  string
	UserID int64
	Reason string
	SetAt  time.Time
}

type UserProfile struct {
	ID        int64
	Username  string
	FirstName string
	LastName  string
	IsBot     bool
	Bio       string
}

type ChatRole struct {
	BotID     string
	ChatID    int64
	UserID    int64
	Role      string
	GrantedBy int64
	GrantedAt time.Time
}

type AntiAbuseSettings struct {
	BotID   string
	ChatID  int64
	Enabled bool
	Action  string
}

type AntiBioSettings struct {
	BotID   string
	ChatID  int64
	Enabled bool
	Action  string
}

type Federation struct {
	ID          string
	BotID       string
	ShortName   string
	DisplayName string
	OwnerUserID int64
	CreatedAt   time.Time
}

type FederationAdmin struct {
	FederationID string
	UserID       int64
	Role         string
	AddedAt      time.Time
}

type FederationBan struct {
	FederationID string
	UserID       int64
	Reason       string
	BannedBy     int64
	BannedAt     time.Time
}

type GlobalBlacklistUser struct {
	BotID     string
	UserID    int64
	Reason    string
	CreatedBy int64
	CreatedAt time.Time
}

type GlobalBlacklistChat struct {
	BotID     string
	ChatID    int64
	Reason    string
	CreatedBy int64
	CreatedAt time.Time
}

type BotStats struct {
	BotID           string
	ChatCount       int
	UserCount       int
	CloneCount      int
	FederationCount int
	JobCount        int
	DeadJobCount    int
}

type Job struct {
	ID            string
	BotID         string
	Kind          string
	Status        string
	RequestedBy   int64
	ReportChatID  int64
	Progress      int
	Total         int
	Attempts      int
	MaxAttempts   int
	PayloadJSON   []byte
	Error         string
	ResultSummary string
	AvailableAt   time.Time
	LockedAt      *time.Time
	LockedBy      string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	CompletedAt   *time.Time
}

type RuntimeBundle struct {
	Settings         ChatSettings
	Moderation       ModerationSettings
	Antiflood        AntifloodSettings
	Captcha          CaptchaSettings
	AntiAbuse        AntiAbuseSettings
	AntiBio          AntiBioSettings
	DisabledCommands map[string]struct{}
	Locks            map[string]LockRule
	Blocklist        []BlocklistRule
}
