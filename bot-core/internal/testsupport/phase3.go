package testsupport

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/telegram"
)

func (m *MemoryStore) SetBotRole(_ context.Context, botID string, userID int64, role string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.roles[botID]; !ok {
		m.roles[botID] = map[int64][]string{}
	}
	if enabled {
		m.roles[botID][userID] = appendRole(m.roles[botID][userID], role)
		return nil
	}
	var next []string
	for _, current := range m.roles[botID][userID] {
		if current != role {
			next = append(next, current)
		}
	}
	m.roles[botID][userID] = next
	return nil
}

func (m *MemoryStore) ListBotRoleUsers(_ context.Context, botID string, role string) ([]domain.UserProfile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var users []domain.UserProfile
	for userID, roles := range m.roles[botID] {
		for _, current := range roles {
			if current == role {
				if user, ok := m.users[userID]; ok {
					users = append(users, user)
				} else {
					users = append(users, domain.UserProfile{ID: userID})
				}
				break
			}
		}
	}
	sort.Slice(users, func(i, j int) bool { return users[i].ID < users[j].ID })
	return users, nil
}

func (m *MemoryStore) GetChatRoles(_ context.Context, botID string, chatID int64, userID int64) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GetChatRolesCalls++
	return append([]string{}, m.chatRoles[chatKey(botID, chatID)][userID]...), nil
}

func (m *MemoryStore) SetChatRole(_ context.Context, botID string, chatID int64, userID int64, role string, grantedBy int64, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_ = grantedBy
	key := chatKey(botID, chatID)
	if _, ok := m.chatRoles[key]; !ok {
		m.chatRoles[key] = map[int64][]string{}
	}
	if enabled {
		m.chatRoles[key][userID] = appendRole(m.chatRoles[key][userID], role)
		return nil
	}
	var next []string
	for _, current := range m.chatRoles[key][userID] {
		if current != role {
			next = append(next, current)
		}
	}
	m.chatRoles[key][userID] = next
	return nil
}

func (m *MemoryStore) ListChatRoleUsers(_ context.Context, botID string, chatID int64, role string) ([]domain.UserProfile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(botID, chatID)
	var users []domain.UserProfile
	for userID, roles := range m.chatRoles[key] {
		for _, current := range roles {
			if current == role {
				if user, ok := m.users[userID]; ok {
					users = append(users, user)
				} else {
					users = append(users, domain.UserProfile{ID: userID})
				}
				break
			}
		}
	}
	return users, nil
}

func (m *MemoryStore) SetLanguage(_ context.Context, botID string, chatID int64, language string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(botID, chatID)
	settings := m.settings[key]
	settings.Language = language
	m.settings[key] = settings
	return nil
}

func (m *MemoryStore) SetAntiAbuseSettings(_ context.Context, settings domain.AntiAbuseSettings) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.antiabuse[chatKey(settings.BotID, settings.ChatID)] = settings
	return nil
}

func (m *MemoryStore) SetAntiBioSettings(_ context.Context, settings domain.AntiBioSettings) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.antibio[chatKey(settings.BotID, settings.ChatID)] = settings
	return nil
}

func (m *MemoryStore) IsAntiBioExempt(_ context.Context, botID string, chatID int64, userID int64) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.antibioExempt[chatKey(botID, chatID)][userID], nil
}

func (m *MemoryStore) SetAntiBioExemption(_ context.Context, botID string, chatID int64, userID int64, addedBy int64, exempt bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_ = addedBy
	key := chatKey(botID, chatID)
	if _, ok := m.antibioExempt[key]; !ok {
		m.antibioExempt[key] = map[int64]bool{}
	}
	if exempt {
		m.antibioExempt[key][userID] = true
	} else {
		delete(m.antibioExempt[key], userID)
	}
	return nil
}

func (m *MemoryStore) ListAntiBioExemptions(_ context.Context, botID string, chatID int64) ([]domain.UserProfile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(botID, chatID)
	var users []domain.UserProfile
	for userID := range m.antibioExempt[key] {
		if user, ok := m.users[userID]; ok {
			users = append(users, user)
		} else {
			users = append(users, domain.UserProfile{ID: userID})
		}
	}
	sort.Slice(users, func(i, j int) bool { return users[i].ID < users[j].ID })
	return users, nil
}

func (m *MemoryStore) ClaimPendingJobs(_ context.Context, workerID string, limit int) ([]domain.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	var jobs []domain.Job
	for _, job := range m.jobs {
		if (job.Status == "pending" || job.Status == "retry") && !job.AvailableAt.After(now) {
			job.Status = "processing"
			job.Attempts++
			job.LockedBy = workerID
			lockTime := now
			job.LockedAt = &lockTime
			m.jobs[job.ID] = job
			jobs = append(jobs, job)
			if len(jobs) >= limit {
				break
			}
		}
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].CreatedAt.Before(jobs[j].CreatedAt) })
	return jobs, nil
}

func (m *MemoryStore) GetJob(_ context.Context, jobID string) (domain.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[jobID]
	if !ok {
		return domain.Job{}, pgx.ErrNoRows
	}
	return job, nil
}

func (m *MemoryStore) ListRecentJobs(_ context.Context, botID string, limit int) ([]domain.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var jobs []domain.Job
	for _, job := range m.jobs {
		if job.BotID == botID {
			jobs = append(jobs, job)
		}
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].CreatedAt.After(jobs[j].CreatedAt) })
	if len(jobs) > limit {
		jobs = jobs[:limit]
	}
	return jobs, nil
}

func (m *MemoryStore) MarkJobCompleted(_ context.Context, jobID string, resultSummary string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job := m.jobs[jobID]
	job.Status = "completed"
	job.ResultSummary = resultSummary
	now := time.Now()
	job.CompletedAt = &now
	job.LockedAt = nil
	job.LockedBy = ""
	m.jobs[jobID] = job
	return nil
}

func (m *MemoryStore) MarkJobRetry(_ context.Context, jobID string, attempts int, errText string, availableAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job := m.jobs[jobID]
	job.Status = "retry"
	job.Attempts = attempts
	job.Error = errText
	job.AvailableAt = availableAt
	job.LockedAt = nil
	job.LockedBy = ""
	m.jobs[jobID] = job
	return nil
}

func (m *MemoryStore) MarkJobDead(_ context.Context, jobID string, errText string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job := m.jobs[jobID]
	job.Status = "dead"
	job.Error = errText
	now := time.Now()
	job.CompletedAt = &now
	job.LockedAt = nil
	job.LockedBy = ""
	m.jobs[jobID] = job
	return nil
}

func (m *MemoryStore) ListChats(_ context.Context, botID string) ([]telegram.Chat, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var chats []telegram.Chat
	prefix := botID + ":"
	for key, chat := range m.chats {
		if strings.HasPrefix(key, prefix) {
			chats = append(chats, chat)
		}
	}
	sort.Slice(chats, func(i, j int) bool { return chats[i].ID < chats[j].ID })
	return chats, nil
}

func (m *MemoryStore) GetStats(_ context.Context, botID string) (domain.BotStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	stats := domain.BotStats{BotID: botID}
	for key := range m.chats {
		if strings.HasPrefix(key, botID+":") {
			stats.ChatCount++
		}
	}
	stats.UserCount = len(m.users)
	for _, bot := range m.botsByID {
		if !bot.IsPrimary {
			stats.CloneCount++
		}
	}
	for _, federation := range m.federations {
		if federation.BotID == botID {
			stats.FederationCount++
		}
	}
	for _, job := range m.jobs {
		if job.BotID == botID {
			stats.JobCount++
			if job.Status == "dead" {
				stats.DeadJobCount++
			}
		}
	}
	return stats, nil
}

func (m *MemoryStore) SetGlobalBlacklistUser(_ context.Context, entry domain.GlobalBlacklistUser, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := userKey(entry.BotID, 0, entry.UserID)
	if enabled {
		m.globalUsers[key] = entry
	} else {
		delete(m.globalUsers, key)
	}
	return nil
}

func (m *MemoryStore) GetGlobalBlacklistUser(_ context.Context, botID string, userID int64) (domain.GlobalBlacklistUser, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.globalUsers[userKey(botID, 0, userID)]
	return entry, ok, nil
}

func (m *MemoryStore) ListGlobalBlacklistUsers(_ context.Context, botID string) ([]domain.GlobalBlacklistUser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.GlobalBlacklistUser
	prefix := botID + ":0:"
	for key, entry := range m.globalUsers {
		if strings.HasPrefix(key, prefix) {
			result = append(result, entry)
		}
	}
	return result, nil
}

func (m *MemoryStore) SetGlobalBlacklistChat(_ context.Context, entry domain.GlobalBlacklistChat, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(entry.BotID, entry.ChatID)
	if enabled {
		m.globalChats[key] = entry
	} else {
		delete(m.globalChats, key)
	}
	return nil
}

func (m *MemoryStore) GetGlobalBlacklistChat(_ context.Context, botID string, chatID int64) (domain.GlobalBlacklistChat, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.globalChats[chatKey(botID, chatID)]
	return entry, ok, nil
}

func (m *MemoryStore) ListGlobalBlacklistChats(_ context.Context, botID string) ([]domain.GlobalBlacklistChat, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.GlobalBlacklistChat
	for key, entry := range m.globalChats {
		if strings.HasPrefix(key, botID+":") {
			result = append(result, entry)
		}
	}
	return result, nil
}

func (m *MemoryStore) CreateFederation(_ context.Context, federation domain.Federation) (domain.Federation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.federations[federation.ID] = federation
	if _, ok := m.federationAdmins[federation.ID]; !ok {
		m.federationAdmins[federation.ID] = map[int64]domain.FederationAdmin{}
	}
	m.federationAdmins[federation.ID][federation.OwnerUserID] = domain.FederationAdmin{
		FederationID: federation.ID,
		UserID:       federation.OwnerUserID,
		Role:         "owner",
	}
	return federation, nil
}

func (m *MemoryStore) DeleteFederation(_ context.Context, federationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.federations, federationID)
	delete(m.federationAdmins, federationID)
	delete(m.federationBans, federationID)
	delete(m.federationChats, federationID)
	return nil
}

func (m *MemoryStore) RenameFederation(_ context.Context, federationID string, shortName string, displayName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	federation, ok := m.federations[federationID]
	if !ok {
		return pgx.ErrNoRows
	}
	federation.ShortName = strings.ToLower(strings.TrimSpace(shortName))
	federation.DisplayName = strings.TrimSpace(displayName)
	m.federations[federationID] = federation
	return nil
}

func (m *MemoryStore) GetFederationByID(_ context.Context, federationID string) (domain.Federation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	federation, ok := m.federations[federationID]
	if !ok {
		return domain.Federation{}, pgx.ErrNoRows
	}
	return federation, nil
}

func (m *MemoryStore) GetFederationByShortName(_ context.Context, botID string, shortName string) (domain.Federation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	shortName = strings.ToLower(shortName)
	for _, federation := range m.federations {
		if federation.BotID == botID && strings.ToLower(federation.ShortName) == shortName {
			return federation, nil
		}
	}
	return domain.Federation{}, pgx.ErrNoRows
}

func (m *MemoryStore) GetFederationByChat(_ context.Context, botID string, chatID int64) (domain.Federation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(botID, chatID)
	for federationID, chats := range m.federationChats {
		for _, chatKeyValue := range chats {
			if chatKeyValue == key {
				return m.federations[federationID], nil
			}
		}
	}
	return domain.Federation{}, pgx.ErrNoRows
}

func (m *MemoryStore) ListFederationsForUser(_ context.Context, botID string, userID int64) ([]domain.Federation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.Federation
	for _, federation := range m.federations {
		if federation.BotID != botID {
			continue
		}
		if federation.OwnerUserID == userID {
			result = append(result, federation)
			continue
		}
		if _, ok := m.federationAdmins[federation.ID][userID]; ok {
			result = append(result, federation)
		}
	}
	return result, nil
}

func (m *MemoryStore) JoinFederation(_ context.Context, federationID string, botID string, chatID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(botID, chatID)
	m.federationChats[federationID] = append(m.federationChats[federationID], key)
	return nil
}

func (m *MemoryStore) LeaveFederation(_ context.Context, federationID string, botID string, chatID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := chatKey(botID, chatID)
	var filtered []string
	for _, current := range m.federationChats[federationID] {
		if current != key {
			filtered = append(filtered, current)
		}
	}
	m.federationChats[federationID] = filtered
	return nil
}

func (m *MemoryStore) ListFederationChats(_ context.Context, federationID string) ([]telegram.Chat, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var chats []telegram.Chat
	for _, key := range m.federationChats[federationID] {
		if chat, ok := m.chats[key]; ok {
			chats = append(chats, chat)
		}
	}
	return chats, nil
}

func (m *MemoryStore) SetFederationAdmin(_ context.Context, federationID string, userID int64, role string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.federationAdmins[federationID]; !ok {
		m.federationAdmins[federationID] = map[int64]domain.FederationAdmin{}
	}
	if enabled {
		m.federationAdmins[federationID][userID] = domain.FederationAdmin{FederationID: federationID, UserID: userID, Role: role}
	} else {
		delete(m.federationAdmins[federationID], userID)
	}
	return nil
}

func (m *MemoryStore) ListFederationAdmins(_ context.Context, federationID string) ([]domain.FederationAdmin, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var admins []domain.FederationAdmin
	for _, admin := range m.federationAdmins[federationID] {
		admins = append(admins, admin)
	}
	return admins, nil
}

func (m *MemoryStore) TransferFederation(_ context.Context, federationID string, newOwnerUserID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	federation := m.federations[federationID]
	federation.OwnerUserID = newOwnerUserID
	m.federations[federationID] = federation
	if _, ok := m.federationAdmins[federationID]; !ok {
		m.federationAdmins[federationID] = map[int64]domain.FederationAdmin{}
	}
	m.federationAdmins[federationID][newOwnerUserID] = domain.FederationAdmin{FederationID: federationID, UserID: newOwnerUserID, Role: "owner"}
	return nil
}

func (m *MemoryStore) SetFederationBan(_ context.Context, ban domain.FederationBan, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.federationBans[ban.FederationID]; !ok {
		m.federationBans[ban.FederationID] = map[int64]domain.FederationBan{}
	}
	if enabled {
		m.federationBans[ban.FederationID][ban.UserID] = ban
	} else {
		delete(m.federationBans[ban.FederationID], ban.UserID)
	}
	return nil
}

func (m *MemoryStore) GetFederationBan(_ context.Context, federationID string, userID int64) (domain.FederationBan, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ban, ok := m.federationBans[federationID][userID]
	return ban, ok, nil
}

func (m *MemoryStore) ExportUserData(_ context.Context, botID string, userID int64) (map[string]any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	export := map[string]any{
		"user":               m.users[userID],
		"warnings":           m.warnings[userKey(botID, 0, userID)],
		"afk":                m.afk[afkKey(botID, userID)],
		"antibio_exemptions": m.antibioExempt,
	}
	return export, nil
}

func (m *MemoryStore) DeleteUserData(_ context.Context, botID string, userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.afk, afkKey(botID, userID))
	delete(m.warnings, userKey(botID, 0, userID))
	for key := range m.approvals {
		delete(m.approvals[key], userID)
	}
	for key := range m.antibioExempt {
		delete(m.antibioExempt[key], userID)
	}
	for key := range m.chatRoles {
		var next []string
		for _, role := range m.chatRoles[key][userID] {
			if role != "mod" {
				next = append(next, role)
			}
		}
		m.chatRoles[key][userID] = next
	}
	var nextRoles []string
	for _, role := range m.roles[botID][userID] {
		if role != "sudo" {
			nextRoles = append(nextRoles, role)
		}
	}
	m.roles[botID][userID] = nextRoles
	return nil
}
