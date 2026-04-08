package permissions

import (
	"context"

	"sukoon/bot-core/internal/persistence"
	"sukoon/bot-core/internal/runtime"
	"sukoon/bot-core/internal/telegram"
)

type Service struct {
	store persistence.Store
}

func New(store persistence.Store) *Service {
	return &Service{store: store}
}

func (s *Service) Load(ctx context.Context, botID string, actorID int64, chatID int64, chatType string, client telegram.Client) (runtime.ActorPermissions, error) {
	perms := runtime.ActorPermissions{}

	roles, err := s.store.GetBotRoles(ctx, botID, actorID)
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

	admins, err := client.GetChatAdministrators(ctx, chatID)
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
