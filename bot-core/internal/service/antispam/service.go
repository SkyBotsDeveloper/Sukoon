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
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), formatAntifloodStatus(rt.RuntimeBundle.Antiflood), rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	if isAntifloodOff(rt.Command.Args[0]) {
		settings := normalizeAntifloodSettings(rt.RuntimeBundle.Antiflood, rt.Bot.ID, rt.ChatID())
		settings.Limit = 0
		settings.Enabled = settings.TimedLimit > 0
		if err := rt.Store.SetAntiflood(ctx, settings); err != nil {
			return err
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Consecutive antiflood disabled.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}

	limit, err := strconv.Atoi(rt.Command.Args[0])
	if err != nil || limit < 2 {
		return fmt.Errorf("flood limit must be at least 2")
	}
	settings := normalizeAntifloodSettings(rt.RuntimeBundle.Antiflood, rt.Bot.ID, rt.ChatID())
	settings.Enabled = true
	settings.Limit = limit
	if err := rt.Store.SetAntiflood(ctx, settings); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Antiflood will trigger after %d consecutive messages.", settings.Limit), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) setFloodMode(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	if len(rt.Command.Args) == 0 {
		settings := normalizeAntifloodSettings(rt.RuntimeBundle.Antiflood, rt.Bot.ID, rt.ChatID())
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Antiflood action is "+formatAntifloodAction(settings.Action, settings.ActionDurationSeconds)+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	mode, durationSeconds, err := parseAntifloodAction(rt.Command.Args)
	if err != nil {
		return err
	}
	settings := normalizeAntifloodSettings(rt.RuntimeBundle.Antiflood, rt.Bot.ID, rt.ChatID())
	settings.Action = mode
	settings.ActionDurationSeconds = durationSeconds
	if err := rt.Store.SetAntiflood(ctx, settings); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Antiflood action set to %s.", formatAntifloodAction(mode, durationSeconds)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) setFloodTimer(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	if len(rt.Command.Args) == 0 {
		return fmt.Errorf("usage: /setfloodtimer <count> <duration>")
	}
	if isAntifloodOff(rt.Command.Args[0]) {
		settings := normalizeAntifloodSettings(rt.RuntimeBundle.Antiflood, rt.Bot.ID, rt.ChatID())
		settings.TimedLimit = 0
		settings.WindowSeconds = 0
		settings.Enabled = settings.Limit > 0
		if err := rt.Store.SetAntiflood(ctx, settings); err != nil {
			return err
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Timed antiflood disabled.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	if len(rt.Command.Args) < 2 {
		return fmt.Errorf("usage: /setfloodtimer <count> <duration>")
	}
	count, err := strconv.Atoi(rt.Command.Args[0])
	if err != nil || count < 2 {
		return fmt.Errorf("timed antiflood count must be at least 2")
	}
	duration, err := parseFlexibleDuration(rt.Command.Args[1])
	if err != nil || duration < time.Second {
		return fmt.Errorf("invalid timed antiflood duration")
	}
	settings := normalizeAntifloodSettings(rt.RuntimeBundle.Antiflood, rt.Bot.ID, rt.ChatID())
	settings.Enabled = true
	settings.TimedLimit = count
	settings.WindowSeconds = int(duration.Seconds())
	if err := rt.Store.SetAntiflood(ctx, settings); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Timed antiflood will trigger after %d messages in %s.", settings.TimedLimit, humanizeFlexibleDuration(duration)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) clearFlood(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	if len(rt.Command.Args) == 0 {
		state := "off"
		if rt.RuntimeBundle.Antiflood.ClearAll {
			state = "on"
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Clearflood is "+state+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	enabled, err := parseAntifloodToggle(rt.Command.Args[0])
	if err != nil {
		return err
	}
	settings := normalizeAntifloodSettings(rt.RuntimeBundle.Antiflood, rt.Bot.ID, rt.ChatID())
	settings.ClearAll = enabled
	if err := rt.Store.SetAntiflood(ctx, settings); err != nil {
		return err
	}
	text := "Clearflood is now off. Sukoon will delete only the messages after the flood limit is reached."
	if enabled {
		text = "Clearflood is now on. Sukoon will delete the full triggered flood set."
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) checkLocks(ctx context.Context, rt *runtime.Context) (bool, error) {
	for lockType, lock := range rt.RuntimeBundle.Locks {
		if matchesLock(lockType, rt.Message) {
			if err := enforceAction(ctx, rt, lock.Action, 0, "lock:"+lockType, true); err != nil {
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
			if err := enforceAction(ctx, rt, rule.Action, 0, "blocklist:"+rule.Pattern, true); err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) checkFlood(ctx context.Context, rt *runtime.Context) (bool, error) {
	settings := normalizeAntifloodSettings(rt.RuntimeBundle.Antiflood, rt.Bot.ID, rt.ChatID())
	if !settings.Enabled || rt.Message == nil || rt.Message.From == nil {
		return false, nil
	}
	window := time.Duration(settings.WindowSeconds) * time.Second
	if window <= 0 {
		window = 10 * time.Second
	}
	tracked, err := rt.State.TrackFlood(ctx, rt.Bot.ID, rt.ChatID(), rt.ActorID(), rt.Message.MessageID, window)
	if err != nil {
		return false, err
	}
	triggered := false
	var deleteIDs []int64
	if settings.Limit > 0 && int(tracked.ConsecutiveCount) > settings.Limit {
		triggered = true
		deleteIDs = append(deleteIDs, antifloodDeletionIDs(tracked.ConsecutiveMessageIDs, settings.Limit, settings.ClearAll)...)
	}
	if settings.TimedLimit > 0 && int(tracked.TimedCount) > settings.TimedLimit {
		triggered = true
		deleteIDs = append(deleteIDs, antifloodDeletionIDs(tracked.TimedMessageIDs, settings.TimedLimit, settings.ClearAll)...)
	}
	if !triggered {
		return false, nil
	}
	deleteFloodMessages(ctx, rt, uniqueMessageIDs(deleteIDs))
	if err := enforceAction(ctx, rt, settings.Action, settings.ActionDurationSeconds, "antiflood", false); err != nil {
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

func enforceAction(ctx context.Context, rt *runtime.Context, action string, durationSeconds int, reason string, deleteCurrent bool) error {
	if deleteCurrent && rt.Message != nil {
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
	case "tban":
		if durationSeconds <= 0 {
			return fmt.Errorf("temporary antiflood ban duration is not configured")
		}
		until := time.Now().Add(time.Duration(durationSeconds) * time.Second)
		if err := rt.Client.BanChatMember(ctx, rt.ChatID(), rt.ActorID(), &until, true); err != nil {
			return err
		}
	case "mute":
		if err := rt.Client.RestrictChatMember(ctx, rt.ChatID(), rt.ActorID(), telegram.RestrictPermissions{CanSendMessages: false}, nil); err != nil {
			return err
		}
	case "tmute":
		if durationSeconds <= 0 {
			return fmt.Errorf("temporary antiflood mute duration is not configured")
		}
		until := time.Now().Add(time.Duration(durationSeconds) * time.Second)
		if err := rt.Client.RestrictChatMember(ctx, rt.ChatID(), rt.ActorID(), telegram.RestrictPermissions{CanSendMessages: false}, &until); err != nil {
			return err
		}
	}
	_ = serviceutil.SendLog(ctx, rt, fmt.Sprintf("antispam: actor=%d action=%s reason=%s", rt.ActorID(), action, reason))
	return nil
}

func normalizeAntifloodSettings(settings domain.AntifloodSettings, botID string, chatID int64) domain.AntifloodSettings {
	settings.BotID = botID
	settings.ChatID = chatID
	if settings.Action == "" {
		settings.Action = "mute"
	}
	if settings.Limit < 0 {
		settings.Limit = 0
	}
	if settings.TimedLimit < 0 {
		settings.TimedLimit = 0
	}
	if settings.WindowSeconds < 0 {
		settings.WindowSeconds = 0
	}
	settings.Enabled = settings.Limit > 0 || settings.TimedLimit > 0
	return settings
}

func formatAntifloodStatus(settings domain.AntifloodSettings) string {
	settings = normalizeAntifloodSettings(settings, settings.BotID, settings.ChatID)
	consecutive := "off"
	if settings.Limit > 0 {
		consecutive = fmt.Sprintf("%d messages in a row", settings.Limit)
	}
	timed := "off"
	if settings.TimedLimit > 0 && settings.WindowSeconds > 0 {
		timed = fmt.Sprintf("%d messages in %s", settings.TimedLimit, humanizeFlexibleDuration(time.Duration(settings.WindowSeconds)*time.Second))
	}
	clearMode := "off"
	if settings.ClearAll {
		clearMode = "on"
	}
	return strings.Join([]string{
		"Antiflood settings:",
		fmt.Sprintf("- Consecutive limit: %s", consecutive),
		fmt.Sprintf("- Timed limit: %s", timed),
		fmt.Sprintf("- Action: %s", formatAntifloodAction(settings.Action, settings.ActionDurationSeconds)),
		fmt.Sprintf("- Clearflood: %s", clearMode),
	}, "\n")
}

func formatAntifloodAction(action string, durationSeconds int) string {
	action = strings.ToLower(strings.TrimSpace(action))
	if (action == "tban" || action == "tmute") && durationSeconds > 0 {
		return fmt.Sprintf("%s %s", action, humanizeFlexibleDuration(time.Duration(durationSeconds)*time.Second))
	}
	if action == "" {
		return "mute"
	}
	return action
}

func parseAntifloodAction(args []string) (string, int, error) {
	mode := strings.ToLower(strings.TrimSpace(args[0]))
	switch mode {
	case "mute", "ban", "kick":
		return mode, 0, nil
	case "tban", "tmute":
		if len(args) < 2 {
			return "", 0, fmt.Errorf("%s requires a duration like 10m or 3d", mode)
		}
		duration, err := parseFlexibleDuration(args[1])
		if err != nil || duration <= 0 {
			return "", 0, fmt.Errorf("invalid %s duration", mode)
		}
		return mode, int(duration.Seconds()), nil
	default:
		return "", 0, fmt.Errorf("flood mode must be ban, mute, kick, tban, or tmute")
	}
}

func parseFlexibleDuration(value string) (time.Duration, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return 0, fmt.Errorf("duration is required")
	}
	if duration, err := time.ParseDuration(value); err == nil {
		return duration, nil
	}
	unit := value[len(value)-1]
	amount, err := strconv.Atoi(value[:len(value)-1])
	if err != nil || amount <= 0 {
		return 0, fmt.Errorf("invalid duration")
	}
	switch unit {
	case 'd':
		return time.Duration(amount) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(amount) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid duration")
	}
}

func humanizeFlexibleDuration(duration time.Duration) string {
	if duration <= 0 {
		return "0s"
	}
	switch {
	case duration%(7*24*time.Hour) == 0:
		return fmt.Sprintf("%dw", int(duration/(7*24*time.Hour)))
	case duration%(24*time.Hour) == 0:
		return fmt.Sprintf("%dd", int(duration/(24*time.Hour)))
	case duration%time.Hour == 0:
		return fmt.Sprintf("%dh", int(duration/time.Hour))
	case duration%time.Minute == 0:
		return fmt.Sprintf("%dm", int(duration/time.Minute))
	default:
		return fmt.Sprintf("%ds", int(duration/time.Second))
	}
}

func parseAntifloodToggle(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "yes", "true":
		return true, nil
	case "off", "no", "false":
		return false, nil
	default:
		return false, fmt.Errorf("value must be yes/no/on/off")
	}
}

func isAntifloodOff(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "0", "off", "no":
		return true
	default:
		return false
	}
}

func antifloodDeletionIDs(messageIDs []int64, limit int, clearAll bool) []int64 {
	if len(messageIDs) == 0 {
		return nil
	}
	if clearAll {
		return append([]int64{}, messageIDs...)
	}
	if limit < len(messageIDs) {
		return append([]int64{}, messageIDs[limit:]...)
	}
	return append([]int64{}, messageIDs[len(messageIDs)-1:]...)
}

func uniqueMessageIDs(messageIDs []int64) []int64 {
	if len(messageIDs) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(messageIDs))
	out := make([]int64, 0, len(messageIDs))
	for _, id := range messageIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func deleteFloodMessages(ctx context.Context, rt *runtime.Context, messageIDs []int64) {
	for _, messageID := range messageIDs {
		_ = rt.Client.DeleteMessage(ctx, rt.ChatID(), messageID)
	}
}
