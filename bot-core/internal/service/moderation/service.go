package moderation

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"sukoon/bot-core/internal/runtime"
	"sukoon/bot-core/internal/serviceutil"
	"sukoon/bot-core/internal/telegram"
)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) Handle(ctx context.Context, rt *runtime.Context) (bool, error) {
	switch rt.Command.Name {
	case "ban":
		return true, s.ban(ctx, rt, false, false)
	case "unban":
		return true, s.unban(ctx, rt)
	case "tban":
		return true, s.tban(ctx, rt)
	case "mute":
		return true, s.mute(ctx, rt, nil, false, false)
	case "unmute":
		return true, s.unmute(ctx, rt)
	case "tmute":
		return true, s.tmute(ctx, rt)
	case "smute":
		return true, s.mute(ctx, rt, nil, true, false)
	case "dmute":
		return true, s.mute(ctx, rt, nil, false, true)
	case "kick":
		return true, s.kick(ctx, rt, false, false)
	case "dkick":
		return true, s.kick(ctx, rt, false, true)
	case "skick":
		return true, s.kick(ctx, rt, true, false)
	case "warn":
		return true, s.warn(ctx, rt)
	case "warns":
		return true, s.warns(ctx, rt)
	case "resetwarns":
		return true, s.resetWarns(ctx, rt)
	case "setwarnlimit":
		return true, s.setWarnLimit(ctx, rt)
	case "setwarnmode":
		return true, s.setWarnMode(ctx, rt)
	default:
		return false, nil
	}
}

func (s *Service) ensureRestrictPerm(rt *runtime.Context) error {
	if !rt.ActorPermissions.CanRestrictMembers {
		return fmt.Errorf("you need restrict permission for this command")
	}
	return nil
}

func (s *Service) ensureMutePerm(rt *runtime.Context) error {
	if !rt.ActorPermissions.CanMuteMembers && !rt.ActorPermissions.CanRestrictMembers {
		return fmt.Errorf("you need mute permission for this command")
	}
	return nil
}

func (s *Service) ban(ctx context.Context, rt *runtime.Context, silent bool, deleteCommand bool) error {
	if err := s.ensureRestrictPerm(rt); err != nil {
		return err
	}
	target, reason, err := parseTargetAndReason(ctx, rt, rt.Command.Args)
	if err != nil {
		return err
	}

	if err := rt.Client.BanChatMember(ctx, rt.ChatID(), target.UserID, nil, true); err != nil {
		return err
	}
	if deleteCommand {
		_ = rt.Client.DeleteMessage(ctx, rt.ChatID(), rt.Message.MessageID)
	}
	if !silent {
		_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Banned %s. %s", target.Name, suffixReason(reason)), rt.ReplyOptions(telegram.SendMessageOptions{}))
		if err != nil {
			return err
		}
	}
	_ = serviceutil.SendLog(ctx, rt, fmt.Sprintf("ban: actor=%d target=%d reason=%s", rt.ActorID(), target.UserID, reason))
	return nil
}

func (s *Service) unban(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureRestrictPerm(rt); err != nil {
		return err
	}
	target, _, err := parseTargetAndReason(ctx, rt, rt.Command.Args)
	if err != nil {
		return err
	}
	if err := rt.Client.UnbanChatMember(ctx, rt.ChatID(), target.UserID, false); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Unbanned %s.", target.Name), rt.ReplyOptions(telegram.SendMessageOptions{}))
	_ = serviceutil.SendLog(ctx, rt, fmt.Sprintf("unban: actor=%d target=%d", rt.ActorID(), target.UserID))
	return err
}

func (s *Service) tban(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureRestrictPerm(rt); err != nil {
		return err
	}
	target, until, reason, err := parseTimedTarget(ctx, rt, rt.Command.Args)
	if err != nil {
		return err
	}
	if err := rt.Client.BanChatMember(ctx, rt.ChatID(), target.UserID, &until, true); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Temp-banned %s until %s. %s", target.Name, until.Format(time.RFC3339), suffixReason(reason)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	_ = serviceutil.SendLog(ctx, rt, fmt.Sprintf("tban: actor=%d target=%d until=%s reason=%s", rt.ActorID(), target.UserID, until.Format(time.RFC3339), reason))
	return err
}

func (s *Service) mute(ctx context.Context, rt *runtime.Context, until *time.Time, silent bool, deleteCommand bool) error {
	if err := s.ensureMutePerm(rt); err != nil {
		return err
	}
	target, reason, err := parseTargetAndReason(ctx, rt, rt.Command.Args)
	if err != nil {
		return err
	}
	if err := rt.Client.RestrictChatMember(ctx, rt.ChatID(), target.UserID, telegram.RestrictPermissions{CanSendMessages: false}, until); err != nil {
		return err
	}
	if deleteCommand {
		_ = rt.Client.DeleteMessage(ctx, rt.ChatID(), rt.Message.MessageID)
	}
	if !silent {
		text := fmt.Sprintf("Muted %s. %s", target.Name, suffixReason(reason))
		if until != nil {
			text = fmt.Sprintf("Muted %s until %s. %s", target.Name, until.Format(time.RFC3339), suffixReason(reason))
		}
		_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
		if err != nil {
			return err
		}
	}
	_ = serviceutil.SendLog(ctx, rt, fmt.Sprintf("mute: actor=%d target=%d until=%v reason=%s", rt.ActorID(), target.UserID, until, reason))
	return nil
}

func (s *Service) tmute(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureMutePerm(rt); err != nil {
		return err
	}
	target, until, reason, err := parseTimedTarget(ctx, rt, rt.Command.Args)
	if err != nil {
		return err
	}
	if err := rt.Client.RestrictChatMember(ctx, rt.ChatID(), target.UserID, telegram.RestrictPermissions{CanSendMessages: false}, &until); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Temp-muted %s until %s. %s", target.Name, until.Format(time.RFC3339), suffixReason(reason)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	_ = serviceutil.SendLog(ctx, rt, fmt.Sprintf("tmute: actor=%d target=%d until=%s reason=%s", rt.ActorID(), target.UserID, until.Format(time.RFC3339), reason))
	return err
}

func (s *Service) unmute(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureMutePerm(rt); err != nil {
		return err
	}
	target, _, err := parseTargetAndReason(ctx, rt, rt.Command.Args)
	if err != nil {
		return err
	}
	if err := rt.Client.RestrictChatMember(ctx, rt.ChatID(), target.UserID, telegram.RestrictPermissions{CanSendMessages: true}, nil); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Unmuted %s.", target.Name), rt.ReplyOptions(telegram.SendMessageOptions{}))
	_ = serviceutil.SendLog(ctx, rt, fmt.Sprintf("unmute: actor=%d target=%d", rt.ActorID(), target.UserID))
	return err
}

func (s *Service) kick(ctx context.Context, rt *runtime.Context, silent bool, deleteCommand bool) error {
	if err := s.ensureRestrictPerm(rt); err != nil {
		return err
	}
	target, reason, err := parseTargetAndReason(ctx, rt, rt.Command.Args)
	if err != nil {
		return err
	}
	now := time.Now().Add(30 * time.Second)
	if err := rt.Client.BanChatMember(ctx, rt.ChatID(), target.UserID, &now, true); err != nil {
		return err
	}
	if err := rt.Client.UnbanChatMember(ctx, rt.ChatID(), target.UserID, true); err != nil {
		return err
	}
	if deleteCommand {
		_ = rt.Client.DeleteMessage(ctx, rt.ChatID(), rt.Message.MessageID)
	}
	if !silent {
		_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Kicked %s. %s", target.Name, suffixReason(reason)), rt.ReplyOptions(telegram.SendMessageOptions{}))
		if err != nil {
			return err
		}
	}
	_ = serviceutil.SendLog(ctx, rt, fmt.Sprintf("kick: actor=%d target=%d reason=%s", rt.ActorID(), target.UserID, reason))
	return nil
}

func (s *Service) warn(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureRestrictPerm(rt); err != nil {
		return err
	}
	target, reason, err := parseTargetAndReason(ctx, rt, rt.Command.Args)
	if err != nil {
		return err
	}
	count, err := rt.Store.IncrementWarnings(ctx, rt.Bot.ID, rt.ChatID(), target.UserID, reason)
	if err != nil {
		return err
	}

	text := fmt.Sprintf("%s now has %d warning(s).", target.Name, count)
	if rt.RuntimeBundle.Moderation.WarnLimit > 0 && count >= rt.RuntimeBundle.Moderation.WarnLimit {
		switch strings.ToLower(rt.RuntimeBundle.Moderation.WarnMode) {
		case "ban":
			if err := rt.Client.BanChatMember(ctx, rt.ChatID(), target.UserID, nil, true); err != nil {
				return err
			}
			text = fmt.Sprintf("%s hit %d warnings and was banned.", target.Name, count)
		case "kick":
			until := time.Now().Add(30 * time.Second)
			if err := rt.Client.BanChatMember(ctx, rt.ChatID(), target.UserID, &until, true); err != nil {
				return err
			}
			if err := rt.Client.UnbanChatMember(ctx, rt.ChatID(), target.UserID, true); err != nil {
				return err
			}
			text = fmt.Sprintf("%s hit %d warnings and was kicked.", target.Name, count)
		default:
			if err := rt.Client.RestrictChatMember(ctx, rt.ChatID(), target.UserID, telegram.RestrictPermissions{CanSendMessages: false}, nil); err != nil {
				return err
			}
			text = fmt.Sprintf("%s hit %d warnings and was muted.", target.Name, count)
		}
	}

	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	_ = serviceutil.SendLog(ctx, rt, fmt.Sprintf("warn: actor=%d target=%d count=%d mode=%s reason=%s", rt.ActorID(), target.UserID, count, rt.RuntimeBundle.Moderation.WarnMode, reason))
	return err
}

func (s *Service) warns(ctx context.Context, rt *runtime.Context) error {
	target, _, err := parseTargetAndReason(ctx, rt, rt.Command.Args)
	if err != nil {
		return err
	}
	count, err := rt.Store.GetWarnings(ctx, rt.Bot.ID, rt.ChatID(), target.UserID)
	if err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("%s has %d warning(s).", target.Name, count), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) resetWarns(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureRestrictPerm(rt); err != nil {
		return err
	}
	target, _, err := parseTargetAndReason(ctx, rt, rt.Command.Args)
	if err != nil {
		return err
	}
	if err := rt.Store.ResetWarnings(ctx, rt.Bot.ID, rt.ChatID(), target.UserID); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Reset warnings for %s.", target.Name), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) setWarnLimit(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureRestrictPerm(rt); err != nil {
		return err
	}
	if len(rt.Command.Args) == 0 {
		return fmt.Errorf("usage: /setwarnlimit <number>")
	}
	limit, err := strconv.Atoi(rt.Command.Args[0])
	if err != nil || limit < 1 {
		return fmt.Errorf("warn limit must be a positive integer")
	}
	if err := rt.Store.SetWarnConfig(ctx, rt.Bot.ID, rt.ChatID(), limit, rt.RuntimeBundle.Moderation.WarnMode); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Warn limit set to %d.", limit), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) setWarnMode(ctx context.Context, rt *runtime.Context) error {
	if err := s.ensureRestrictPerm(rt); err != nil {
		return err
	}
	if len(rt.Command.Args) == 0 {
		return fmt.Errorf("usage: /setwarnmode <mute|kick|ban>")
	}
	mode := strings.ToLower(rt.Command.Args[0])
	switch mode {
	case "mute", "kick", "ban":
	default:
		return fmt.Errorf("warn mode must be mute, kick, or ban")
	}
	if err := rt.Store.SetWarnConfig(ctx, rt.Bot.ID, rt.ChatID(), rt.RuntimeBundle.Moderation.WarnLimit, mode); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Warn mode set to %s.", mode), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func parseTargetAndReason(ctx context.Context, rt *runtime.Context, args []string) (serviceutil.Target, string, error) {
	target, err := serviceutil.ResolveTarget(ctx, rt, args)
	if err != nil {
		return serviceutil.Target{}, "", err
	}
	if rt.Message != nil && rt.Message.ReplyToMessage != nil {
		return target, strings.TrimSpace(strings.Join(args, " ")), nil
	}
	if len(args) <= 1 {
		return target, "", nil
	}
	return target, strings.TrimSpace(strings.Join(args[1:], " ")), nil
}

func parseTimedTarget(ctx context.Context, rt *runtime.Context, args []string) (serviceutil.Target, time.Time, string, error) {
	if rt.Message != nil && rt.Message.ReplyToMessage != nil {
		if len(args) == 0 {
			return serviceutil.Target{}, time.Time{}, "", fmt.Errorf("usage: reply and pass a duration")
		}
		duration, err := time.ParseDuration(args[0])
		if err != nil {
			return serviceutil.Target{}, time.Time{}, "", fmt.Errorf("invalid duration")
		}
		target, err := serviceutil.ResolveTarget(ctx, rt, args)
		if err != nil {
			return serviceutil.Target{}, time.Time{}, "", err
		}
		return target, time.Now().Add(duration), strings.TrimSpace(strings.Join(args[1:], " ")), nil
	}

	if len(args) < 2 {
		return serviceutil.Target{}, time.Time{}, "", fmt.Errorf("usage: /command <user_id> <duration> [reason]")
	}
	target, err := serviceutil.ResolveTarget(ctx, rt, args[:1])
	if err != nil {
		return serviceutil.Target{}, time.Time{}, "", err
	}
	duration, err := time.ParseDuration(args[1])
	if err != nil {
		return serviceutil.Target{}, time.Time{}, "", fmt.Errorf("invalid duration")
	}
	return target, time.Now().Add(duration), strings.TrimSpace(strings.Join(args[2:], " ")), nil
}

func suffixReason(reason string) string {
	if strings.TrimSpace(reason) == "" {
		return ""
	}
	return "Reason: " + reason
}
