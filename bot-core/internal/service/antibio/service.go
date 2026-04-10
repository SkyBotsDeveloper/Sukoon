package antibio

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"sukoon/bot-core/internal/runtime"
	"sukoon/bot-core/internal/serviceutil"
	"sukoon/bot-core/internal/telegram"
)

var bioLinkMatcher = regexp.MustCompile(`(?i)(https?://|t\.me/|telegram\.me/|wa\.me/|discord\.gg/|@[\pL\pN_]{5,})`)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) HandleCommand(ctx context.Context, rt *runtime.Context) (bool, error) {
	switch rt.Command.Name {
	case "antibio":
		return true, s.antibio(ctx, rt)
	case "free":
		return true, s.free(ctx, rt, true)
	case "unfree":
		return true, s.free(ctx, rt, false)
	case "freelist":
		return true, s.freeList(ctx, rt)
	default:
		return false, nil
	}
}

func (s *Service) HandleMessage(ctx context.Context, rt *runtime.Context) (bool, error) {
	if !rt.RuntimeBundle.AntiBio.Enabled || rt.Message == nil || rt.Message.From == nil {
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
	exempt, err := rt.Store.IsAntiBioExempt(ctx, rt.Bot.ID, rt.ChatID(), rt.ActorID())
	if err != nil {
		return false, err
	}
	if exempt {
		return false, nil
	}
	leaseKey := fmt.Sprintf("antibio:%s:%d:%d", rt.Bot.ID, rt.ChatID(), rt.ActorID())
	acquired, err := rt.State.AcquireLease(ctx, leaseKey, 30*time.Second)
	if err != nil {
		return false, err
	}
	if !acquired {
		return false, nil
	}
	chat, err := rt.Client.GetChat(ctx, rt.ActorID())
	if err != nil {
		_ = rt.State.DeleteLease(ctx, leaseKey)
		return false, nil
	}
	if !bioLinkMatcher.MatchString(strings.TrimSpace(chat.Bio)) {
		_ = rt.State.SetLease(ctx, leaseKey, 6*time.Hour)
		return false, nil
	}
	if err := serviceutil.EnforceUserAction(ctx, rt, rt.ActorID(), rt.RuntimeBundle.AntiBio.Action, "antibio", rt.Message.MessageID); err != nil {
		_ = rt.State.DeleteLease(ctx, leaseKey)
		return false, err
	}
	_ = rt.State.SetLease(ctx, leaseKey, 6*time.Hour)
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "User bio policy triggered.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	if err != nil {
		return false, err
	}
	_ = serviceutil.SendLogCategory(ctx, rt, serviceutil.LogCategoryAutomated, fmt.Sprintf("antibio: actor=%d action=%s", rt.ActorID(), rt.RuntimeBundle.AntiBio.Action))
	return true, nil
}

func (s *Service) antibio(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	if len(rt.Command.Args) == 0 {
		status := "off"
		if rt.RuntimeBundle.AntiBio.Enabled {
			status = "on"
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "AntiBio is "+status+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	enabled, err := parseToggle(rt.Command.Args[0])
	if err != nil {
		return err
	}
	settings := rt.RuntimeBundle.AntiBio
	settings.BotID = rt.Bot.ID
	settings.ChatID = rt.ChatID()
	settings.Enabled = enabled
	if len(rt.Command.Args) > 1 {
		settings.Action = strings.ToLower(rt.Command.Args[1])
	}
	if settings.Action == "" {
		settings.Action = "kick"
	}
	if err := rt.Store.SetAntiBioSettings(ctx, settings); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "AntiBio updated.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) free(ctx context.Context, rt *runtime.Context, enabled bool) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	target, _, err := targetAndReason(ctx, rt)
	if err != nil {
		return err
	}
	if err := rt.Store.SetAntiBioExemption(ctx, rt.Bot.ID, rt.ChatID(), target.UserID, rt.ActorID(), enabled); err != nil {
		return err
	}
	text := "AntiBio exemption added for " + target.Name + "."
	if !enabled {
		text = "AntiBio exemption removed for " + target.Name + "."
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) freeList(ctx context.Context, rt *runtime.Context) error {
	users, err := rt.Store.ListAntiBioExemptions(ctx, rt.Bot.ID, rt.ChatID())
	if err != nil {
		return err
	}
	if len(users) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "No AntiBio exemptions.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	parts := make([]string, 0, len(users))
	for _, user := range users {
		parts = append(parts, serviceutil.DisplayNameFromProfile(user))
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "AntiBio exemptions: "+strings.Join(parts, ", "), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
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

func targetAndReason(ctx context.Context, rt *runtime.Context) (serviceutil.Target, string, error) {
	target, err := serviceutil.ResolveTarget(ctx, rt, rt.Command.Args)
	if err != nil {
		return serviceutil.Target{}, "", err
	}
	if rt.Message != nil && rt.Message.ReplyToMessage != nil {
		return target, strings.TrimSpace(strings.Join(rt.Command.Args, " ")), nil
	}
	if len(rt.Command.Args) <= 1 {
		return target, "", nil
	}
	return target, strings.TrimSpace(strings.Join(rt.Command.Args[1:], " ")), nil
}
