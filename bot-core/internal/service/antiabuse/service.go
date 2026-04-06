package antiabuse

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/runtime"
	"sukoon/bot-core/internal/serviceutil"
	"sukoon/bot-core/internal/telegram"
)

var abuseMatcher = regexp.MustCompile(`(?i)\b(?:asshole|bastard|bitch|bsdk|chutiya|harami|madarchod|motherfucker|bhenchod)\b`)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) HandleCommand(ctx context.Context, rt *runtime.Context) (bool, error) {
	if rt.Command.Name != "antiabuse" {
		return false, nil
	}
	if !rt.ActorPermissions.IsChatAdmin {
		return true, fmt.Errorf("admin rights required")
	}
	if len(rt.Command.Args) == 0 {
		status := "off"
		if rt.RuntimeBundle.AntiAbuse.Enabled {
			status = "on"
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Antiabuse is "+status+".", telegram.SendMessageOptions{})
		return true, err
	}
	enabled, err := parseToggle(rt.Command.Args[0])
	if err != nil {
		return true, err
	}
	settings := rt.RuntimeBundle.AntiAbuse
	settings.BotID = rt.Bot.ID
	settings.ChatID = rt.ChatID()
	settings.Enabled = enabled
	if len(rt.Command.Args) > 1 {
		settings.Action = strings.ToLower(rt.Command.Args[1])
	}
	if settings.Action == "" {
		settings.Action = "delete_warn"
	}
	if err := rt.Store.SetAntiAbuseSettings(ctx, settings); err != nil {
		return true, err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Antiabuse updated.", telegram.SendMessageOptions{})
	return true, err
}

func (s *Service) HandleMessage(ctx context.Context, rt *runtime.Context) (bool, error) {
	if !rt.RuntimeBundle.AntiAbuse.Enabled || rt.Message == nil || rt.Message.From == nil {
		return false, nil
	}
	if rt.ActorPermissions.IsOwner || rt.ActorPermissions.IsSudo || rt.ActorPermissions.IsChatAdmin {
		return false, nil
	}
	approved, err := rt.Store.IsApproved(ctx, rt.Bot.ID, rt.ChatID(), rt.ActorID())
	if err != nil {
		return false, err
	}
	if approved {
		return false, nil
	}

	text := strings.TrimSpace(rt.Text())
	if text == "" || !abuseMatcher.MatchString(text) {
		return false, nil
	}

	if err := serviceutil.EnforceUserAction(ctx, rt, rt.ActorID(), rt.RuntimeBundle.AntiAbuse.Action, "antiabuse", rt.Message.MessageID); err != nil {
		return false, err
	}

	notice := "Abusive message removed."
	if strings.EqualFold(rt.RuntimeBundle.AntiAbuse.Action, "warn") || strings.EqualFold(rt.RuntimeBundle.AntiAbuse.Action, "delete_warn") {
		count, err := rt.Store.IncrementWarnings(ctx, rt.Bot.ID, rt.ChatID(), rt.ActorID(), "antiabuse")
		if err != nil {
			return false, err
		}
		notice = fmt.Sprintf("Abusive message removed. Warning count: %d.", count)
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), notice, telegram.SendMessageOptions{})
	if err != nil {
		return false, err
	}
	_ = serviceutil.SendLog(ctx, rt, fmt.Sprintf("antiabuse: actor=%d action=%s", rt.ActorID(), rt.RuntimeBundle.AntiAbuse.Action))
	return true, nil
}

func parseToggle(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "enable", "enabled":
		return true, nil
	case "off", "disable", "disabled":
		return false, nil
	default:
		return false, fmt.Errorf("expected on or off")
	}
}

func DefaultSettings(botID string, chatID int64) domain.AntiAbuseSettings {
	return domain.AntiAbuseSettings{
		BotID:   botID,
		ChatID:  chatID,
		Enabled: false,
		Action:  "delete_warn",
	}
}
