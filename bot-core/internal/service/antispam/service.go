package antispam

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/runtime"
	"sukoon/bot-core/internal/serviceutil"
	"sukoon/bot-core/internal/telegram"
)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) HandleCommand(ctx context.Context, rt *runtime.Context) (bool, error) {
	switch rt.Command.Name {
	case "lock":
		return true, s.lock(ctx, rt, true)
	case "unlock":
		return true, s.lock(ctx, rt, false)
	case "locks":
		return true, s.listLocks(ctx, rt)
	case "locktypes":
		return true, s.lockTypes(ctx, rt)
	case "addblocklist":
		return true, s.addBlocklist(ctx, rt)
	case "rmbl", "rmblocklist":
		return true, s.removeBlocklist(ctx, rt)
	case "unblocklistall":
		return true, s.removeAllBlocklist(ctx, rt)
	case "blocklist":
		return true, s.listBlocklist(ctx, rt)
	case "setflood", "flood":
		return true, s.setFlood(ctx, rt)
	case "setfloodmode", "floodmode":
		return true, s.setFloodMode(ctx, rt)
	case "setfloodtimer":
		return true, s.setFloodTimer(ctx, rt)
	case "clearflood":
		return true, s.clearFlood(ctx, rt)
	default:
		return false, nil
	}
}

func (s *Service) HandleMessage(ctx context.Context, rt *runtime.Context) (bool, error) {
	if rt.Message == nil || rt.Message.From == nil {
		return false, nil
	}
	if rt.ActorPermissions.IsChatAdmin {
		return false, nil
	}

	approved, err := rt.Store.IsApproved(ctx, rt.Bot.ID, rt.ChatID(), rt.ActorID())
	if err != nil {
		return false, err
	}
	if approved {
		return false, nil
	}

	if handled, err := s.checkLocks(ctx, rt); handled || err != nil {
		return handled, err
	}
	if handled, err := s.checkBlocklist(ctx, rt); handled || err != nil {
		return handled, err
	}
	if handled, err := s.checkFlood(ctx, rt); handled || err != nil {
		return handled, err
	}
	return false, nil
}

func (s *Service) lock(ctx context.Context, rt *runtime.Context, enable bool) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	if len(rt.Command.Args) == 0 {
		return fmt.Errorf("usage: /lock <type>")
	}
	lockType := canonicalLockType(rt.Command.Args[0])
	if lockType == "" || !slices.Contains(supportedLockTypes(), lockType) {
		return fmt.Errorf("unsupported lock type")
	}
	if enable {
		err := rt.Store.UpsertLock(ctx, domain.LockRule{
			BotID:    rt.Bot.ID,
			ChatID:   rt.ChatID(),
			LockType: lockType,
			Action:   "delete",
		})
		if err != nil {
			return err
		}
		_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Locked %s.", lockType), rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}

	if err := rt.Store.DeleteLock(ctx, rt.Bot.ID, rt.ChatID(), lockType); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Unlocked %s.", lockType), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) listLocks(ctx context.Context, rt *runtime.Context) error {
	if len(rt.RuntimeBundle.Locks) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "No active locks.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	keys := make([]string, 0, len(rt.RuntimeBundle.Locks))
	for lockType := range rt.RuntimeBundle.Locks {
		keys = append(keys, lockType)
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Active locks: "+strings.Join(keys, ", "), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) lockTypes(ctx context.Context, rt *runtime.Context) error {
	text := "Supported lock types: links, forwards, media, sticker, gif."
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) addBlocklist(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	if len(rt.Command.Args) < 2 {
		return fmt.Errorf("usage: /addblocklist <word|phrase|regex> <pattern>")
	}
	matchMode := strings.ToLower(rt.Command.Args[0])
	pattern := strings.TrimSpace(strings.Join(rt.Command.Args[1:], " "))
	if pattern == "" {
		return fmt.Errorf("pattern is required")
	}
	switch matchMode {
	case "word":
		matchMode = "word"
	case "phrase", "contains":
		matchMode = "contains"
	case "regex":
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("invalid regex: %w", err)
		}
	default:
		return fmt.Errorf("match mode must be word, phrase, or regex")
	}

	rule, err := rt.Store.AddBlocklistRule(ctx, domain.BlocklistRule{
		BotID:     rt.Bot.ID,
		ChatID:    rt.ChatID(),
		Pattern:   pattern,
		MatchMode: matchMode,
		Action:    "delete",
		CreatedBy: rt.ActorID(),
	})
	if err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Added blocklist rule #%d for %q.", rule.ID, pattern), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) removeBlocklist(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	items := serviceutil.SplitBulkItems(rt.Command.RawArgs)
	if len(items) == 0 {
		return fmt.Errorf("usage: /rmbl <pattern>")
	}
	for _, pattern := range items {
		if err := rt.Store.DeleteBlocklistRule(ctx, rt.Bot.ID, rt.ChatID(), pattern); err != nil {
			return err
		}
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Removed %d blocklist rule(s).", len(items)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) removeAllBlocklist(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	rules, err := rt.Store.ListBlocklistRules(ctx, rt.Bot.ID, rt.ChatID())
	if err != nil {
		return err
	}
	if len(rules) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "No blocklist rules to remove.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	for _, rule := range rules {
		if err := rt.Store.DeleteBlocklistRule(ctx, rt.Bot.ID, rt.ChatID(), rule.Pattern); err != nil {
			return err
		}
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Removed %d blocklist rule(s).", len(rules)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) listBlocklist(ctx context.Context, rt *runtime.Context) error {
	if len(rt.RuntimeBundle.Blocklist) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "No blocklist rules.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	lines := make([]string, 0, len(rt.RuntimeBundle.Blocklist))
	for _, rule := range rt.RuntimeBundle.Blocklist {
		lines = append(lines, fmt.Sprintf("%d. [%s] %s", rule.ID, rule.MatchMode, rule.Pattern))
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), strings.Join(lines, "\n"), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) setFlood(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	if len(rt.Command.Args) == 0 {
		if !rt.RuntimeBundle.Antiflood.Enabled {
			_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Antiflood is off.", rt.ReplyOptions(telegram.SendMessageOptions{}))
			return err
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Antiflood is on: limit=%d window=%ds action=%s.", rt.RuntimeBundle.Antiflood.Limit, rt.RuntimeBundle.Antiflood.WindowSeconds, rt.RuntimeBundle.Antiflood.Action), rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	if strings.EqualFold(rt.Command.Args[0], "off") {
		settings := rt.RuntimeBundle.Antiflood
		settings.Enabled = false
		settings.BotID = rt.Bot.ID
		settings.ChatID = rt.ChatID()
		if err := rt.Store.SetAntiflood(ctx, settings); err != nil {
			return err
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Antiflood disabled.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}

	limit, err := strconv.Atoi(rt.Command.Args[0])
	if err != nil || limit < 2 {
		return fmt.Errorf("flood limit must be at least 2")
	}
	settings := rt.RuntimeBundle.Antiflood
	settings.BotID = rt.Bot.ID
	settings.ChatID = rt.ChatID()
	settings.Enabled = true
	settings.Limit = limit
	if settings.WindowSeconds == 0 {
		settings.WindowSeconds = 10
	}
	if settings.Action == "" {
		settings.Action = "mute"
	}
	if err := rt.Store.SetAntiflood(ctx, settings); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Antiflood enabled at %d messages per %d seconds.", settings.Limit, settings.WindowSeconds), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) setFloodMode(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	if len(rt.Command.Args) == 0 {
		action := rt.RuntimeBundle.Antiflood.Action
		if action == "" {
			action = "mute"
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Antiflood action is "+action+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	mode := strings.ToLower(rt.Command.Args[0])
	switch mode {
	case "mute", "ban", "kick":
	default:
		return fmt.Errorf("flood mode must be mute, ban, or kick")
	}
	settings := rt.RuntimeBundle.Antiflood
	settings.BotID = rt.Bot.ID
	settings.ChatID = rt.ChatID()
	settings.Enabled = true
	if settings.Limit == 0 {
		settings.Limit = 6
	}
	if settings.WindowSeconds == 0 {
		settings.WindowSeconds = 10
	}
	settings.Action = mode
	if err := rt.Store.SetAntiflood(ctx, settings); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Antiflood action set to %s.", mode), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) setFloodTimer(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	if len(rt.Command.Args) == 0 {
		return fmt.Errorf("usage: /setfloodtimer <seconds>")
	}
	seconds, err := strconv.Atoi(rt.Command.Args[0])
	if err != nil || seconds < 1 {
		return fmt.Errorf("flood timer must be at least 1 second")
	}
	settings := rt.RuntimeBundle.Antiflood
	settings.BotID = rt.Bot.ID
	settings.ChatID = rt.ChatID()
	settings.WindowSeconds = seconds
	if settings.Limit == 0 {
		settings.Limit = 6
	}
	if settings.Action == "" {
		settings.Action = "mute"
	}
	if err := rt.Store.SetAntiflood(ctx, settings); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Antiflood timer set to %d seconds.", seconds), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) clearFlood(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	targetID := rt.ActorID()
	targetName := "yourself"
	if (rt.Message != nil && rt.Message.ReplyToMessage != nil) || len(rt.Command.Args) > 0 {
		target, err := serviceutil.ResolveTarget(ctx, rt, rt.Command.Args)
		if err != nil {
			return err
		}
		targetID = target.UserID
		targetName = target.Name
	}
	if err := rt.State.ClearFlood(ctx, rt.Bot.ID, rt.ChatID(), targetID); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Cleared antiflood history for "+targetName+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) checkLocks(ctx context.Context, rt *runtime.Context) (bool, error) {
	for lockType, lock := range rt.RuntimeBundle.Locks {
		if matchesLock(lockType, rt.Message) {
			if err := enforceAction(ctx, rt, lock.Action, "lock:"+lockType); err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) checkBlocklist(ctx context.Context, rt *runtime.Context) (bool, error) {
	text := strings.ToLower(strings.TrimSpace(rt.Text()))
	if text == "" {
		return false, nil
	}
	for _, rule := range rt.RuntimeBundle.Blocklist {
		if matchesBlocklist(rule, text) {
			if err := enforceAction(ctx, rt, rule.Action, "blocklist:"+rule.Pattern); err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) checkFlood(ctx context.Context, rt *runtime.Context) (bool, error) {
	if !rt.RuntimeBundle.Antiflood.Enabled || rt.Message == nil || rt.Message.From == nil {
		return false, nil
	}
	window := time.Duration(rt.RuntimeBundle.Antiflood.WindowSeconds) * time.Second
	if window <= 0 {
		window = 10 * time.Second
	}
	count, err := rt.State.TrackFlood(ctx, rt.Bot.ID, rt.ChatID(), rt.ActorID(), rt.Message.MessageID, window)
	if err != nil {
		return false, err
	}
	if int(count) <= rt.RuntimeBundle.Antiflood.Limit {
		return false, nil
	}
	if err := enforceAction(ctx, rt, rt.RuntimeBundle.Antiflood.Action, "antiflood"); err != nil {
		return false, err
	}
	_ = rt.State.ClearFlood(ctx, rt.Bot.ID, rt.ChatID(), rt.ActorID())
	return true, nil
}

func matchesLock(lockType string, message *telegram.Message) bool {
	switch lockType {
	case "links":
		text := strings.ToLower(message.Text + " " + message.Caption)
		return strings.Contains(text, "http://") || strings.Contains(text, "https://") || strings.Contains(text, "t.me/")
	case "forwards":
		return message.ForwardOrigin != nil
	case "media":
		return len(message.Photo) > 0 || message.Video != nil || message.Document != nil || message.Animation != nil
	case "sticker":
		return message.Sticker != nil
	case "gif":
		return message.Animation != nil
	default:
		return false
	}
}

func supportedLockTypes() []string {
	return []string{"links", "forwards", "media", "sticker", "gif"}
}

func canonicalLockType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "link", "links", "url", "urls":
		return "links"
	case "forward", "forwards":
		return "forwards"
	case "media":
		return "media"
	case "sticker", "stickers":
		return "sticker"
	case "gif", "gifs", "animation", "animations":
		return "gif"
	default:
		return ""
	}
}

func matchesBlocklist(rule domain.BlocklistRule, text string) bool {
	switch rule.MatchMode {
	case "regex":
		re, err := regexp.Compile(rule.Pattern)
		return err == nil && re.MatchString(text)
	case "contains":
		return strings.Contains(text, strings.ToLower(rule.Pattern))
	default:
		pattern := `\b` + regexp.QuoteMeta(strings.ToLower(rule.Pattern)) + `\b`
		re := regexp.MustCompile(pattern)
		return re.MatchString(text)
	}
}

func enforceAction(ctx context.Context, rt *runtime.Context, action string, reason string) error {
	if rt.Message != nil {
		_ = rt.Client.DeleteMessage(ctx, rt.ChatID(), rt.Message.MessageID)
	}

	switch strings.ToLower(action) {
	case "ban":
		if err := rt.Client.BanChatMember(ctx, rt.ChatID(), rt.ActorID(), nil, true); err != nil {
			return err
		}
	case "kick":
		until := time.Now().Add(30 * time.Second)
		if err := rt.Client.BanChatMember(ctx, rt.ChatID(), rt.ActorID(), &until, true); err != nil {
			return err
		}
		if err := rt.Client.UnbanChatMember(ctx, rt.ChatID(), rt.ActorID(), true); err != nil {
			return err
		}
	case "mute":
		if err := rt.Client.RestrictChatMember(ctx, rt.ChatID(), rt.ActorID(), telegram.RestrictPermissions{CanSendMessages: false}, nil); err != nil {
			return err
		}
	}
	_ = serviceutil.SendLog(ctx, rt, fmt.Sprintf("antispam: actor=%d action=%s reason=%s", rt.ActorID(), action, reason))
	return nil
}
