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
	CleanCommandAll       bool
	CleanCommandAdmin     bool
	CleanCommandUser      bool
	CleanCommandOther     bool
	LogCategorySettings   bool
	LogCategoryAdmin      bool
	LogCategoryUser       bool
	LogCategoryAutomated  bool
	LogCategoryReports    bool
	LogCategoryOther      bool
	LockWarns             bool
	BlocklistAction       string
	BlocklistActionSecs   int
	BlocklistDelete       bool
	BlocklistReason       string
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

type ChatConnection struct {
	BotID        string
	UserID       int64
	ChatID       int64
	ChatType     string
	ChatTitle    string
	ChatUsername string
	ConnectedAt  time.Time
}

type ModerationSettings struct {
	BotID     string
	ChatID    int64
	WarnLimit int
	WarnMode  string
}

type AntifloodSettings struct {
	BotID                 string
	ChatID                int64
	Enabled               bool
	Limit                 int
	TimedLimit            int
	WindowSeconds         int
	Action                string
	ActionDurationSeconds int
	ClearAll              bool
}

type FloodTrackResult struct {
	ConsecutiveCount      int64
	ConsecutiveMessageIDs []int64
	TimedCount            int64
	TimedMessageIDs       []int64
}

type CaptchaSettings struct {
	BotID             string
	ChatID            int64
	Enabled           bool
	Mode              string
	TimeoutSeconds    int
	RulesRequired     bool
	AutoUnmuteSeconds int
	KickOnTimeout     bool
	ButtonText        string
	FailureAction     string
	ChallengeDigits   int
}

type LockRule struct {
	BotID                 string
	ChatID                int64
	LockType              string
	Action                string
	ActionDurationSeconds int
	Reason                string
}

type LockAllowlistEntry struct {
	BotID     string
	ChatID    int64
	Item      string
	CreatedAt time.Time
}

type BlocklistRule struct {
	ID                    int64
	BotID                 string
	ChatID                int64
	Pattern               string
	MatchMode             string
	Action                string
	ActionDurationSeconds int
	DeleteBehavior        string
	Reason                string
	CreatedBy             int64
	CreatedAt             time.Time
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
	Mode          string
	RulesRequired bool
	RulesAccepted bool
	TimeoutAction string
	FailureAction string
}

type AFKState struct {
	BotID  string
	UserID int64
	Reason string
	SetAt  time.Time
}

type Approval struct {
	BotID      string
	ChatID     int64
	UserID     int64
	ApprovedBy int64
	Reason     string
	ApprovedAt time.Time
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

type AntiRaidSettings struct {
	BotID                 string
	ChatID                int64
	EnabledUntil          *time.Time
	RaidDurationSeconds   int
	ActionDurationSeconds int
	AutoThreshold         int
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
	AntiRaid         AntiRaidSettings
	Captcha          CaptchaSettings
	AntiAbuse        AntiAbuseSettings
	AntiBio          AntiBioSettings
	DisabledCommands map[string]struct{}
	Locks            map[string]LockRule
	LockAllowlist    []string
	Blocklist        []BlocklistRule
}

func (s ChatSettings) CleanCommandCategoryEnabled(category string) bool {
	if s.CleanCommandAll {
		return true
	}
	switch category {
	case "all":
		return s.CleanCommandAll
	case "admin":
		return s.CleanCommandAdmin
	case "user":
		return s.CleanCommandUser
	case "other":
		return s.CleanCommandOther
	default:
		return false
	}
}

func (s ChatSettings) EnabledCleanCommandCategories() []string {
	enabled := make([]string, 0, 4)
	for _, category := range []string{"all", "admin", "user", "other"} {
		if s.CleanCommandCategoryEnabled(category) {
			enabled = append(enabled, category)
		}
	}
	return enabled
}

func (s ChatSettings) LogCategoryEnabled(category string) bool {
	switch category {
	case "settings":
		return s.LogCategorySettings
	case "admin":
		return s.LogCategoryAdmin
	case "user":
		return s.LogCategoryUser
	case "automated":
		return s.LogCategoryAutomated
	case "reports":
		return s.LogCategoryReports
	case "other":
		return s.LogCategoryOther
	case "all":
		return s.LogCategorySettings && s.LogCategoryAdmin && s.LogCategoryUser && s.LogCategoryAutomated && s.LogCategoryReports && s.LogCategoryOther
	default:
		return false
	}
}

func (s ChatSettings) EnabledLogCategories() []string {
	enabled := make([]string, 0, 6)
	for _, category := range []string{"settings", "admin", "user", "automated", "reports", "other"} {
		if s.LogCategoryEnabled(category) {
			enabled = append(enabled, category)
		}
	}
	return enabled
}
