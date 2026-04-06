package serviceutil

import (
	"context"
	"fmt"
	"strings"
	"time"

	"sukoon/bot-core/internal/runtime"
	"sukoon/bot-core/internal/telegram"
)

func EnforceUserAction(ctx context.Context, rt *runtime.Context, userID int64, action string, reason string, deleteMessageID int64) error {
	if deleteMessageID != 0 {
		_ = rt.Client.DeleteMessage(ctx, rt.ChatID(), deleteMessageID)
	}

	switch strings.ToLower(strings.TrimSpace(action)) {
	case "ban":
		if err := rt.Client.BanChatMember(ctx, rt.ChatID(), userID, nil, true); err != nil {
			return err
		}
	case "kick":
		until := time.Now().Add(30 * time.Second)
		if err := rt.Client.BanChatMember(ctx, rt.ChatID(), userID, &until, true); err != nil {
			return err
		}
		if err := rt.Client.UnbanChatMember(ctx, rt.ChatID(), userID, true); err != nil {
			return err
		}
	case "mute":
		if err := rt.Client.RestrictChatMember(ctx, rt.ChatID(), userID, telegram.RestrictPermissions{CanSendMessages: false}, nil); err != nil {
			return err
		}
	case "delete", "warn", "delete_warn", "":
	default:
		return fmt.Errorf("unsupported moderation action %s", action)
	}
	return nil
}
