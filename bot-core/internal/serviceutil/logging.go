package serviceutil

import (
	"context"

	"sukoon/bot-core/internal/runtime"
	"sukoon/bot-core/internal/telegram"
)

func SendLog(ctx context.Context, rt *runtime.Context, text string) error {
	if rt.RuntimeBundle.Settings.LogChannelID == nil {
		return nil
	}
	_, err := rt.Client.SendMessage(ctx, *rt.RuntimeBundle.Settings.LogChannelID, text, telegram.SendMessageOptions{})
	return err
}
