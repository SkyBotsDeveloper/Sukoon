package serviceutil

import (
	"context"

	"sukoon/bot-core/internal/runtime"
	"sukoon/bot-core/internal/telegram"
)

const (
	LogCategorySettings  = "settings"
	LogCategoryAdmin     = "admin"
	LogCategoryUser      = "user"
	LogCategoryAutomated = "automated"
	LogCategoryReports   = "reports"
	LogCategoryOther     = "other"
)

func SendLog(ctx context.Context, rt *runtime.Context, text string) error {
	return SendLogCategory(ctx, rt, LogCategoryOther, text)
}

func SendLogCategory(ctx context.Context, rt *runtime.Context, category string, text string) error {
	if rt.RuntimeBundle.Settings.LogChannelID == nil {
		return nil
	}
	if !rt.RuntimeBundle.Settings.LogCategoryEnabled(category) {
		return nil
	}
	_, err := rt.Client.SendMessage(ctx, *rt.RuntimeBundle.Settings.LogChannelID, text, telegram.SendMessageOptions{})
	return err
}
