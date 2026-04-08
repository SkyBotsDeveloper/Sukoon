package permissions

import (
	"context"
	"fmt"
	"sync"
	"time"

	"sukoon/bot-core/internal/persistence"
	"sukoon/bot-core/internal/runtime"
	"sukoon/bot-core/internal/telegram"
)

type Service struct {
	store persistence.Store
	mu    sync.RWMutex

	botRoleCache map[string]cachedRoles
	adminCache   map[string]cachedAdmins
}

func New(store persistence.Store) *Service {
	return &Service{
		store:        store,
		botRoleCache: map[string]cachedRoles{},
		adminCache:   map[string]cachedAdmins{},
	}
}

func (s *Service) Load(ctx context.Context, botID string, actorID int64, chatID int64, chatType string, client telegram.Client) (runtime.ActorPermissions, error) {
	perms := runtime.ActorPermissions{}

	roles, err := s.getBotRoles(ctx, botID, actorID)
	if err != nil {
		return perms, err
	}
	for _, role := range roles {
		switch role {
		case "owner":
			perms.IsOwner = true
		case "sudo":
			perms.IsSudo = true
		}
	}

	if perms.IsOwner || perms.IsSudo {
		perms.IsChatAdmin = true
		perms.CanDeleteMessages = true
		perms.CanMuteMembers = true
		perms.CanRestrictMembers = true
		perms.CanChangeInfo = true
		perms.CanPinMessages = true
		perms.CanPromoteMembers = true
		return perms, nil
	}

	if chatType == "private" {
		return perms, nil
	}

	admins, err := s.ChatAdministrators(ctx, botID, chatID, client)
	if err != nil {
		return perms, err
	}
	for _, admin := range admins {
		if admin.User.ID != actorID {
			continue
		}
		perms.IsChatAdmin = true
		perms.IsChatCreator = admin.Status == "creator"
		perms.CanDeleteMessages = admin.CanDeleteMessages || admin.Status == "creator"
		perms.CanMuteMembers = admin.CanRestrictMembers || admin.Status == "creator"
		perms.CanRestrictMembers = admin.CanRestrictMembers || admin.Status == "creator"
		perms.CanChangeInfo = admin.CanChangeInfo || admin.Status == "creator"
		perms.CanPinMessages = admin.CanPinMessages || admin.Status == "creator"
		perms.CanPromoteMembers = admin.CanPromoteMembers || admin.Status == "creator"
		break
	}

	chatRoles, err := s.store.GetChatRoles(ctx, botID, chatID, actorID)
	if err != nil {
		return perms, err
	}
	for _, role := range chatRoles {
		switch role {
		case "mod":
			perms.IsSilentMod = true
			perms.CanDeleteMessages = true
			perms.CanMuteMembers = true
			perms.CanRestrictMembers = true
		case "muter":
			perms.IsSilentMod = true
			perms.CanMuteMembers = true
		}
	}
	return perms, nil
}

func (s *Service) ChatAdministrators(ctx context.Context, botID string, chatID int64, client telegram.Client) ([]telegram.ChatAdministrator, error) {
	key := fmt.Sprintf("%s:%d", botID, chatID)
	now := time.Now()

	s.mu.RLock()
	entry, ok := s.adminCache[key]
	s.mu.RUnlock()
	if ok && now.Before(entry.expiresAt) {
		return append([]telegram.ChatAdministrator{}, entry.admins...), nil
	}

	admins, err := client.GetChatAdministrators(ctx, chatID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.adminCache[key] = cachedAdmins{
		admins:    append([]telegram.ChatAdministrator{}, admins...),
		expiresAt: now.Add(15 * time.Second),
	}
	s.mu.Unlock()
	return admins, nil
}

func (s *Service) RefreshChatAdministrators(ctx context.Context, botID string, chatID int64, client telegram.Client) ([]telegram.ChatAdministrator, error) {
	admins, err := client.GetChatAdministrators(ctx, chatID)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%s:%d", botID, chatID)
	s.mu.Lock()
	s.adminCache[key] = cachedAdmins{
		admins:    append([]telegram.ChatAdministrator{}, admins...),
		expiresAt: time.Now().Add(15 * time.Second),
	}
	s.mu.Unlock()
	return admins, nil
}

func (s *Service) getBotRoles(ctx context.Context, botID string, actorID int64) ([]string, error) {
	key := fmt.Sprintf("%s:%d", botID, actorID)
	now := time.Now()

	s.mu.RLock()
	entry, ok := s.botRoleCache[key]
	s.mu.RUnlock()
	if ok && now.Before(entry.expiresAt) {
		return append([]string{}, entry.roles...), nil
	}

	roles, err := s.store.GetBotRoles(ctx, botID, actorID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.botRoleCache[key] = cachedRoles{
		roles:     append([]string{}, roles...),
		expiresAt: now.Add(30 * time.Second),
	}
	s.mu.Unlock()
	return roles, nil
}

type cachedRoles struct {
	roles     []string
	expiresAt time.Time
}

type cachedAdmins struct {
	admins    []telegram.ChatAdministrator
	expiresAt time.Time
}
