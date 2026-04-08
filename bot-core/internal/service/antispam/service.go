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
	case "blocklistmode":
		return true, s.setBlocklistMode(ctx, rt)
	case "blocklistdelete":
		return true, s.setBlocklistDelete(ctx, rt)
	case "setblocklistreason":
		return true, s.setBlocklistReason(ctx, rt)
	case "resetblocklistreason":
		return true, s.resetBlocklistReason(ctx, rt)
	case "setflood", "flood":
		return true, s.setFlood(ctx, rt)
	case "setfloodmode", "floodmode":
		return true, s.setFloodMode(ctx, rt)
	case "setfloodtimer":
		return true, s.setFloodTimer(ctx, rt)
	case "clearflood":
		return true, s.clearFlood(ctx, rt)
	case "antiraid":
		return true, s.antiRaid(ctx, rt)
	case "raidtime":
		return true, s.raidTime(ctx, rt)
	case "raidactiontime":
		return true, s.raidActionTime(ctx, rt)
	case "autoantiraid":
		return true, s.autoAntiRaid(ctx, rt)
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

func (s *Service) HandleJoin(ctx context.Context, rt *runtime.Context, member telegram.User) (bool, error) {
	settings := normalizeAntiRaidSettings(rt.RuntimeBundle.AntiRaid, rt.Bot.ID, rt.ChatID())
	now := time.Now()

	if antiRaidIsActive(settings, now) {
		if shouldSkipAntiRaid(rt, member.ID) {
			return false, nil
		}
		if err := s.enforceAntiRaidJoin(ctx, rt, member, settings); err != nil {
			return false, err
		}
		return true, nil
	}

	if settings.AutoThreshold <= 0 {
		return false, nil
	}
	count, err := rt.State.TrackJoinBurst(ctx, rt.Bot.ID, rt.ChatID(), member.ID, time.Minute)
	if err != nil {
		return false, err
	}
	if int(count) <= settings.AutoThreshold {
		return false, nil
	}

	enabledUntil := now.Add(time.Duration(settings.RaidDurationSeconds) * time.Second)
	settings.EnabledUntil = &enabledUntil
	if err := rt.Store.SetAntiRaidSettings(ctx, settings); err != nil {
		return false, err
	}
	rt.RuntimeBundle.AntiRaid = settings

	if acquired, err := rt.State.AcquireLease(ctx, fmt.Sprintf("antiraid:auto:%s:%d", rt.Bot.ID, rt.ChatID()), 15*time.Second); err == nil && acquired {
		_, _ = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("AntiRaid auto-enabled for %s after %d joins in under a minute.", humanizeFlexibleDuration(time.Duration(settings.RaidDurationSeconds)*time.Second), count), telegram.SendMessageOptions{})
	}

	if shouldSkipAntiRaid(rt, member.ID) {
		return false, nil
	}
	if err := s.enforceAntiRaidJoin(ctx, rt, member, settings); err != nil {
		return false, err
	}
	return true, nil
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
	rules, err := parseBlocklistRules(rt)
	if err != nil {
		return err
	}
	added := make([]string, 0, len(rules))
	for _, rule := range rules {
		rule.BotID = rt.Bot.ID
		rule.ChatID = rt.ChatID()
		rule.CreatedBy = rt.ActorID()
		if _, err := rt.Store.AddBlocklistRule(ctx, rule); err != nil {
			return err
		}
		added = append(added, rule.Pattern)
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Added %d blocklist rule(s): %s.", len(added), strings.Join(added, ", ")), rt.ReplyOptions(telegram.SendMessageOptions{}))
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
	if !rt.ActorPermissions.IsOwner && !rt.ActorPermissions.IsSudo && !rt.ActorPermissions.IsChatCreator {
		return fmt.Errorf("chat creator rights required")
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
	settings := normalizeBlocklistSettings(rt.RuntimeBundle.Settings)
	lines := []string{
		"Blocklist settings:",
		fmt.Sprintf("- Mode: %s", formatBlocklistAction(settings.Action, settings.ActionDurationSeconds)),
		fmt.Sprintf("- Delete messages: %s", onOff(settings.DeleteMessages)),
	}
	if strings.TrimSpace(settings.DefaultReason) != "" {
		lines = append(lines, "- Default reason: "+settings.DefaultReason)
	}
	if len(rt.RuntimeBundle.Blocklist) == 0 {
		lines = append(lines, "", "No blocklist rules.")
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), strings.Join(lines, "\n"), rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	lines = append(lines, "", "Rules:")
	for _, rule := range rt.RuntimeBundle.Blocklist {
		description := fmt.Sprintf("%d. [%s] %s", rule.ID, rule.MatchMode, rule.Pattern)
		action := effectiveBlocklistAction(rule, settings)
		actionText := formatBlocklistAction(action.Action, action.DurationSeconds)
		if action.Action != "nothing" {
			description += " {" + actionText + "}"
		}
		switch effectiveBlocklistDelete(rule, settings) {
		case true:
			description += " {del}"
		default:
			description += " {nodel}"
		}
		if reason := effectiveBlocklistReason(rule, settings); reason != "" {
			description += " - " + reason
		}
		lines = append(lines, description)
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), strings.Join(lines, "\n"), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) setBlocklistMode(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	settings := normalizeBlocklistSettings(rt.RuntimeBundle.Settings)
	if len(rt.Command.Args) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Blocklist mode is "+formatBlocklistAction(settings.Action, settings.ActionDurationSeconds)+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	action, durationSeconds, err := parseBlocklistAction(rt.Command.Args)
	if err != nil {
		return err
	}
	if err := rt.Store.SetBlocklistMode(ctx, rt.Bot.ID, rt.ChatID(), action, durationSeconds); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Blocklist mode set to "+formatBlocklistAction(action, durationSeconds)+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) setBlocklistDelete(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	settings := normalizeBlocklistSettings(rt.RuntimeBundle.Settings)
	if len(rt.Command.Args) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Blocklist delete is "+onOff(settings.DeleteMessages)+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	enabled, err := parseAntifloodToggle(rt.Command.Args[0])
	if err != nil {
		return fmt.Errorf("value must be yes/no/on/off")
	}
	if err := rt.Store.SetBlocklistDelete(ctx, rt.Bot.ID, rt.ChatID(), enabled); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), "Blocklist delete is now "+onOff(enabled)+".", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) setBlocklistReason(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	reason := strings.TrimSpace(rt.Command.RawArgs)
	if reason == "" {
		return fmt.Errorf("usage: /setblocklistreason <reason>")
	}
	if err := rt.Store.SetBlocklistReason(ctx, rt.Bot.ID, rt.ChatID(), reason); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Default blocklist reason updated.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) resetBlocklistReason(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	if err := rt.Store.SetBlocklistReason(ctx, rt.Bot.ID, rt.ChatID(), ""); err != nil {
		return err
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Default blocklist reason reset.", rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

type blocklistSettings struct {
	Action                string
	ActionDurationSeconds int
	DeleteMessages        bool
	DefaultReason         string
}

type blocklistAction struct {
	Action          string
	DurationSeconds int
}

type blocklistModifiers struct {
	Action                string
	ActionDurationSeconds int
	DeleteBehavior        string
}

func normalizeBlocklistSettings(settings domain.ChatSettings) blocklistSettings {
	action := strings.ToLower(strings.TrimSpace(settings.BlocklistAction))
	if action == "" {
		action = "nothing"
	}
	deleteMessages := settings.BlocklistDelete
	if !settings.BlocklistDelete && settings.BlocklistAction == "" && settings.BlocklistReason == "" && settings.BlocklistActionSecs == 0 {
		deleteMessages = true
	}
	return blocklistSettings{
		Action:                action,
		ActionDurationSeconds: settings.BlocklistActionSecs,
		DeleteMessages:        deleteMessages,
		DefaultReason:         strings.TrimSpace(settings.BlocklistReason),
	}
}

func effectiveBlocklistAction(rule domain.BlocklistRule, settings blocklistSettings) blocklistAction {
	action := strings.ToLower(strings.TrimSpace(rule.Action))
	if action == "" {
		return blocklistAction{Action: settings.Action, DurationSeconds: settings.ActionDurationSeconds}
	}
	return blocklistAction{Action: action, DurationSeconds: rule.ActionDurationSeconds}
}

func effectiveBlocklistDelete(rule domain.BlocklistRule, settings blocklistSettings) bool {
	switch strings.ToLower(strings.TrimSpace(rule.DeleteBehavior)) {
	case "delete", "del":
		return true
	case "nodel", "nodelete":
		return false
	default:
		return settings.DeleteMessages
	}
}

func effectiveBlocklistReason(rule domain.BlocklistRule, settings blocklistSettings) string {
	if strings.TrimSpace(rule.Reason) != "" {
		return strings.TrimSpace(rule.Reason)
	}
	return settings.DefaultReason
}

func formatBlocklistAction(action string, durationSeconds int) string {
	action = strings.ToLower(strings.TrimSpace(action))
	if action == "" {
		action = "nothing"
	}
	if (action == "tban" || action == "tmute") && durationSeconds > 0 {
		return fmt.Sprintf("%s %s", action, humanizeFlexibleDuration(time.Duration(durationSeconds)*time.Second))
	}
	return action
}

func parseBlocklistAction(args []string) (string, int, error) {
	action := strings.ToLower(strings.TrimSpace(args[0]))
	switch action {
	case "nothing", "none", "off":
		return "nothing", 0, nil
	case "ban", "mute", "kick", "warn":
		return action, 0, nil
	case "tban", "tmute":
		if len(args) < 2 {
			return "", 0, fmt.Errorf("%s requires a duration like 6h or 5d", action)
		}
		duration, err := parseFlexibleDuration(args[1])
		if err != nil || duration <= 0 {
			return "", 0, fmt.Errorf("invalid %s duration", action)
		}
		return action, int(duration.Seconds()), nil
	default:
		return "", 0, fmt.Errorf("blocklist mode must be nothing, ban, mute, kick, warn, tban, or tmute")
	}
}

func parseBlocklistRules(rt *runtime.Context) ([]domain.BlocklistRule, error) {
	if rt.Message != nil && rt.Message.ReplyToMessage != nil && rt.Message.ReplyToMessage.Sticker != nil && strings.TrimSpace(rt.Command.RawArgs) == "" {
		if strings.TrimSpace(rt.Message.ReplyToMessage.Sticker.SetName) == "" {
			return nil, fmt.Errorf("reply to a sticker with a known sticker pack")
		}
		return []domain.BlocklistRule{{
			Pattern:        rt.Message.ReplyToMessage.Sticker.SetName,
			MatchMode:      "stickerpack",
			DeleteBehavior: "inherit",
		}}, nil
	}

	raw := strings.TrimSpace(rt.Command.RawArgs)
	if raw == "" {
		return nil, fmt.Errorf("usage: /addblocklist <trigger> <reason>")
	}
	cleaned, modifiers, err := extractBlocklistModifiers(raw)
	if err != nil {
		return nil, err
	}
	trigger, reason, err := splitBlocklistTriggerAndReason(cleaned)
	if err != nil {
		return nil, err
	}
	items, err := expandBlocklistTrigger(trigger, rt.Message)
	if err != nil {
		return nil, err
	}
	rules := make([]domain.BlocklistRule, 0, len(items))
	for _, item := range items {
		matchMode, pattern, err := classifyBlocklistTrigger(item, rt.Message)
		if err != nil {
			return nil, err
		}
		rules = append(rules, domain.BlocklistRule{
			Pattern:               pattern,
			MatchMode:             matchMode,
			Action:                modifiers.Action,
			ActionDurationSeconds: modifiers.ActionDurationSeconds,
			DeleteBehavior:        modifiers.DeleteBehavior,
			Reason:                strings.TrimSpace(reason),
		})
	}
	return rules, nil
}

func extractBlocklistModifiers(raw string) (string, blocklistModifiers, error) {
	modifiers := blocklistModifiers{DeleteBehavior: "inherit"}
	re := regexp.MustCompile(`\{([^{}]+)\}`)
	matches := re.FindAllStringSubmatch(raw, -1)
	for _, match := range matches {
		token := strings.TrimSpace(match[1])
		lower := strings.ToLower(token)
		switch lower {
		case "del":
			modifiers.DeleteBehavior = "delete"
			continue
		case "nodel":
			modifiers.DeleteBehavior = "nodel"
			continue
		}
		fields := strings.Fields(lower)
		if len(fields) == 0 {
			continue
		}
		action, durationSeconds, err := parseBlocklistAction(fields)
		if err != nil {
			return "", blocklistModifiers{}, err
		}
		modifiers.Action = action
		modifiers.ActionDurationSeconds = durationSeconds
	}
	return strings.TrimSpace(re.ReplaceAllString(raw, "")), modifiers, nil
}

func splitBlocklistTriggerAndReason(raw string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", fmt.Errorf("usage: /addblocklist <trigger> <reason>")
	}
	if raw[0] == '"' {
		end := findClosingQuote(raw)
		if end <= 0 {
			return "", "", fmt.Errorf("unterminated quoted blocklist trigger")
		}
		return raw[1:end], strings.TrimSpace(raw[end+1:]), nil
	}
	if raw[0] == '(' {
		end := strings.Index(raw, ")")
		if end <= 0 {
			return "", "", fmt.Errorf("unterminated blocklist group")
		}
		return strings.TrimSpace(raw[:end+1]), strings.TrimSpace(raw[end+1:]), nil
	}
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return "", "", fmt.Errorf("usage: /addblocklist <trigger> <reason>")
	}
	trigger := fields[0]
	reason := strings.TrimSpace(raw[len(trigger):])
	if legacyTrigger, legacyReason, ok := legacyBlocklistTrigger(raw, fields); ok {
		trigger = legacyTrigger
		reason = legacyReason
	}
	return trigger, reason, nil
}

func legacyBlocklistTrigger(raw string, fields []string) (string, string, bool) {
	if len(fields) < 2 {
		return "", "", false
	}
	switch strings.ToLower(fields[0]) {
	case "word":
		reason := strings.TrimSpace(strings.TrimPrefix(raw, strings.Join(fields[:2], " ")))
		return fields[1], reason, true
	case "phrase", "contains":
		return strings.TrimSpace(raw[len(fields[0])+1:]), "", true
	case "regex":
		return "regex:" + strings.TrimSpace(raw[len(fields[0])+1:]), "", true
	default:
		return "", "", false
	}
}

func findClosingQuote(raw string) int {
	escaped := false
	for idx := 1; idx < len(raw); idx++ {
		switch raw[idx] {
		case '\\':
			escaped = !escaped
		case '"':
			if !escaped {
				return idx
			}
			escaped = false
		default:
			escaped = false
		}
	}
	return -1
}

func expandBlocklistTrigger(trigger string, message *telegram.Message) ([]string, error) {
	trigger = strings.TrimSpace(trigger)
	if trigger == "" {
		return nil, fmt.Errorf("blocklist trigger is required")
	}
	if strings.HasPrefix(trigger, "(") && strings.HasSuffix(trigger, ")") {
		inner := strings.TrimSpace(trigger[1 : len(trigger)-1])
		if inner == "" {
			return nil, fmt.Errorf("blocklist group cannot be empty")
		}
		parts := strings.Split(inner, ",")
		items := make([]string, 0, len(parts))
		for _, part := range parts {
			value := strings.TrimSpace(strings.Trim(part, `"`))
			if value != "" {
				items = append(items, value)
			}
		}
		if len(items) == 0 {
			return nil, fmt.Errorf("blocklist group cannot be empty")
		}
		return items, nil
	}
	if strings.EqualFold(trigger, "stickerpack:<>") {
		if message == nil || message.ReplyToMessage == nil || message.ReplyToMessage.Sticker == nil || strings.TrimSpace(message.ReplyToMessage.Sticker.SetName) == "" {
			return nil, fmt.Errorf("reply to a sticker with a known sticker pack to use stickerpack:<>")
		}
		return []string{"stickerpack:" + message.ReplyToMessage.Sticker.SetName}, nil
	}
	return []string{strings.Trim(trigger, `"`)}, nil
}

func classifyBlocklistTrigger(trigger string, message *telegram.Message) (string, string, error) {
	_ = message
	trigger = strings.TrimSpace(strings.Trim(trigger, `"`))
	if trigger == "" {
		return "", "", fmt.Errorf("blocklist trigger is required")
	}
	lower := strings.ToLower(trigger)
	switch {
	case strings.HasPrefix(lower, "regex:"):
		pattern := strings.TrimSpace(trigger[len("regex:"):])
		if _, err := regexp.Compile(pattern); err != nil {
			return "", "", fmt.Errorf("invalid regex: %w", err)
		}
		return "regex", pattern, nil
	case strings.HasPrefix(lower, "exact:"):
		return "exact", strings.TrimSpace(trigger[len("exact:"):]), nil
	case strings.HasPrefix(lower, "prefix:"):
		return "prefix", strings.TrimSpace(trigger[len("prefix:"):]), nil
	case strings.HasPrefix(lower, "file:"):
		return "file", strings.TrimSpace(trigger[len("file:"):]), nil
	case strings.HasPrefix(lower, "inline:"):
		return "inline", strings.TrimSpace(trigger[len("inline:"):]), nil
	case strings.HasPrefix(lower, "forward:"):
		return "forward", strings.TrimSpace(trigger[len("forward:"):]), nil
	case strings.HasPrefix(lower, "lookalike:"):
		return "lookalike", strings.TrimSpace(trigger[len("lookalike:"):]), nil
	case strings.HasPrefix(lower, "stickerpack:"):
		return "stickerpack", strings.TrimSpace(trigger[len("stickerpack:"):]), nil
	case strings.Contains(trigger, "?") || strings.Contains(trigger, "*"):
		return "wildcard", trigger, nil
	case strings.Contains(trigger, " "):
		return "contains", trigger, nil
	default:
		return "word", trigger, nil
	}
}

func onOff(value bool) string {
	if value {
		return "on"
	}
	return "off"
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

func (s *Service) antiRaid(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}

	settings := normalizeAntiRaidSettings(rt.RuntimeBundle.AntiRaid, rt.Bot.ID, rt.ChatID())
	if len(rt.Command.Args) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), formatAntiRaidStatus(settings), rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}

	arg := strings.ToLower(strings.TrimSpace(rt.Command.Args[0]))
	switch arg {
	case "off", "no", "0":
		settings.EnabledUntil = nil
		if err := rt.Store.SetAntiRaidSettings(ctx, settings); err != nil {
			return err
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "AntiRaid disabled.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	case "on", "yes":
		until := time.Now().Add(time.Duration(settings.RaidDurationSeconds) * time.Second)
		settings.EnabledUntil = &until
		if err := rt.Store.SetAntiRaidSettings(ctx, settings); err != nil {
			return err
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("AntiRaid enabled for %s.", humanizeFlexibleDuration(time.Duration(settings.RaidDurationSeconds)*time.Second)), rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	default:
		duration, err := parseFlexibleDuration(rt.Command.Args[0])
		if err != nil || duration <= 0 {
			return fmt.Errorf("usage: /antiraid [on|off|<duration>]")
		}
		until := time.Now().Add(duration)
		settings.EnabledUntil = &until
		if err := rt.Store.SetAntiRaidSettings(ctx, settings); err != nil {
			return err
		}
		_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("AntiRaid enabled for %s.", humanizeFlexibleDuration(duration)), rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
}

func (s *Service) raidTime(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	settings := normalizeAntiRaidSettings(rt.RuntimeBundle.AntiRaid, rt.Bot.ID, rt.ChatID())
	if len(rt.Command.Args) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("AntiRaid duration is %s.", humanizeFlexibleDuration(time.Duration(settings.RaidDurationSeconds)*time.Second)), rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	duration, err := parseFlexibleDuration(rt.Command.Args[0])
	if err != nil || duration <= 0 {
		return fmt.Errorf("usage: /raidtime <duration>")
	}
	settings.RaidDurationSeconds = int(duration.Seconds())
	if err := rt.Store.SetAntiRaidSettings(ctx, settings); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("AntiRaid duration set to %s.", humanizeFlexibleDuration(duration)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) raidActionTime(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	settings := normalizeAntiRaidSettings(rt.RuntimeBundle.AntiRaid, rt.Bot.ID, rt.ChatID())
	if len(rt.Command.Args) == 0 {
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("AntiRaid action time is %s.", humanizeFlexibleDuration(time.Duration(settings.ActionDurationSeconds)*time.Second)), rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	duration, err := parseFlexibleDuration(rt.Command.Args[0])
	if err != nil || duration <= 0 {
		return fmt.Errorf("usage: /raidactiontime <duration>")
	}
	settings.ActionDurationSeconds = int(duration.Seconds())
	if err := rt.Store.SetAntiRaidSettings(ctx, settings); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("AntiRaid action time set to %s.", humanizeFlexibleDuration(duration)), rt.ReplyOptions(telegram.SendMessageOptions{}))
	return err
}

func (s *Service) autoAntiRaid(ctx context.Context, rt *runtime.Context) error {
	if !rt.ActorPermissions.IsChatAdmin {
		return fmt.Errorf("admin rights required")
	}
	settings := normalizeAntiRaidSettings(rt.RuntimeBundle.AntiRaid, rt.Bot.ID, rt.ChatID())
	if len(rt.Command.Args) == 0 {
		if settings.AutoThreshold <= 0 {
			_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Auto AntiRaid is off.", rt.ReplyOptions(telegram.SendMessageOptions{}))
			return err
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Auto AntiRaid will enable if more than %d users join in under a minute.", settings.AutoThreshold), rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	if isAntifloodOff(rt.Command.Args[0]) {
		settings.AutoThreshold = 0
		if err := rt.Store.SetAntiRaidSettings(ctx, settings); err != nil {
			return err
		}
		_, err := rt.Client.SendMessage(ctx, rt.ChatID(), "Auto AntiRaid disabled.", rt.ReplyOptions(telegram.SendMessageOptions{}))
		return err
	}
	threshold, err := strconv.Atoi(rt.Command.Args[0])
	if err != nil || threshold < 1 {
		return fmt.Errorf("auto antiraid threshold must be at least 1")
	}
	settings.AutoThreshold = threshold
	if err := rt.Store.SetAntiRaidSettings(ctx, settings); err != nil {
		return err
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), fmt.Sprintf("Auto AntiRaid will trigger if more than %d users join in under a minute.", threshold), rt.ReplyOptions(telegram.SendMessageOptions{}))
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
	if rt.Message == nil {
		return false, nil
	}
	message := rt.Message
	for _, rule := range rt.RuntimeBundle.Blocklist {
		if matchesBlocklist(rule, message) {
			if err := s.applyBlocklistRule(ctx, rt, rule); err != nil {
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

func normalizeAntiRaidSettings(settings domain.AntiRaidSettings, botID string, chatID int64) domain.AntiRaidSettings {
	settings.BotID = botID
	settings.ChatID = chatID
	if settings.RaidDurationSeconds <= 0 {
		settings.RaidDurationSeconds = 6 * 60 * 60
	}
	if settings.ActionDurationSeconds <= 0 {
		settings.ActionDurationSeconds = 60 * 60
	}
	if settings.AutoThreshold < 0 {
		settings.AutoThreshold = 0
	}
	return settings
}

func antiRaidIsActive(settings domain.AntiRaidSettings, now time.Time) bool {
	return settings.EnabledUntil != nil && settings.EnabledUntil.After(now)
}

func formatAntiRaidStatus(settings domain.AntiRaidSettings) string {
	settings = normalizeAntiRaidSettings(settings, settings.BotID, settings.ChatID)
	status := "off"
	if antiRaidIsActive(settings, time.Now()) {
		status = "on until " + settings.EnabledUntil.Format(time.RFC3339)
	}
	auto := "off"
	if settings.AutoThreshold > 0 {
		auto = fmt.Sprintf("more than %d joins in under a minute", settings.AutoThreshold)
	}
	return strings.Join([]string{
		"AntiRaid settings:",
		fmt.Sprintf("- Status: %s", status),
		fmt.Sprintf("- Raid duration: %s", humanizeFlexibleDuration(time.Duration(settings.RaidDurationSeconds)*time.Second)),
		fmt.Sprintf("- Tempban duration: %s", humanizeFlexibleDuration(time.Duration(settings.ActionDurationSeconds)*time.Second)),
		fmt.Sprintf("- Auto AntiRaid: %s", auto),
	}, "\n")
}

func shouldSkipAntiRaid(rt *runtime.Context, userID int64) bool {
	if userID == 0 {
		return true
	}
	if _, ok := rt.KnownChatAdmins[userID]; ok {
		return true
	}
	roles, err := rt.Store.GetBotRoles(rt.Base, rt.Bot.ID, userID)
	if err != nil {
		return false
	}
	for _, role := range roles {
		if role == "owner" || role == "sudo" {
			return true
		}
	}
	return false
}

func (s *Service) enforceAntiRaidJoin(ctx context.Context, rt *runtime.Context, member telegram.User, settings domain.AntiRaidSettings) error {
	until := time.Now().Add(time.Duration(settings.ActionDurationSeconds) * time.Second)
	if err := rt.Client.BanChatMember(ctx, rt.ChatID(), member.ID, &until, true); err != nil {
		return err
	}
	_ = serviceutil.SendLog(ctx, rt, fmt.Sprintf("antiraid: user=%d until=%s", member.ID, until.Format(time.RFC3339)))
	return nil
}

func matchesBlocklist(rule domain.BlocklistRule, message *telegram.Message) bool {
	text := normalizeBlocklistText(message.Text + " " + message.Caption)
	switch rule.MatchMode {
	case "regex":
		re, err := regexp.Compile(rule.Pattern)
		return err == nil && re.MatchString(text)
	case "contains":
		return strings.Contains(text, normalizeBlocklistText(rule.Pattern))
	case "exact":
		return matchExactPattern(text, normalizeBlocklistText(rule.Pattern))
	case "prefix":
		return matchPrefixPattern(text, normalizeBlocklistText(rule.Pattern))
	case "wildcard":
		return matchWildcardPattern(text, normalizeBlocklistText(rule.Pattern))
	case "file":
		return message.Document != nil && matchWildcardPattern(strings.ToLower(strings.TrimSpace(message.Document.FileName)), strings.ToLower(strings.TrimSpace(rule.Pattern)))
	case "inline":
		return message.ViaBot != nil && matchWildcardPattern(strings.ToLower("@"+strings.TrimPrefix(message.ViaBot.Username, "@")), strings.ToLower(strings.TrimSpace(rule.Pattern)))
	case "forward":
		return matchForwardPattern(message, strings.ToLower(strings.TrimSpace(rule.Pattern)))
	case "lookalike":
		return matchWordPattern(normalizeLookalikeText(text), normalizeLookalikeText(normalizeBlocklistText(rule.Pattern)))
	case "stickerpack":
		return message.Sticker != nil && strings.EqualFold(strings.TrimSpace(message.Sticker.SetName), strings.TrimSpace(rule.Pattern))
	default:
		return matchWordPattern(text, normalizeBlocklistText(rule.Pattern))
	}
}

func (s *Service) applyBlocklistRule(ctx context.Context, rt *runtime.Context, rule domain.BlocklistRule) error {
	settings := normalizeBlocklistSettings(rt.RuntimeBundle.Settings)
	action := effectiveBlocklistAction(rule, settings)
	deleteMessage := effectiveBlocklistDelete(rule, settings)
	reason := effectiveBlocklistReason(rule, settings)
	if deleteMessage && rt.Message != nil {
		_ = rt.Client.DeleteMessage(ctx, rt.ChatID(), rt.Message.MessageID)
	}

	switch action.Action {
	case "", "nothing":
		_ = serviceutil.SendLog(ctx, rt, fmt.Sprintf("blocklist: actor=%d pattern=%s delete=%t", rt.ActorID(), rule.Pattern, deleteMessage))
		return nil
	case "warn":
		return s.warnBlocklistUser(ctx, rt, reason, rule.Pattern)
	case "ban":
		if err := rt.Client.BanChatMember(ctx, rt.ChatID(), rt.ActorID(), nil, true); err != nil {
			return err
		}
		return s.sendBlocklistActionMessage(ctx, rt, "Banned", reason)
	case "kick":
		until := time.Now().Add(30 * time.Second)
		if err := rt.Client.BanChatMember(ctx, rt.ChatID(), rt.ActorID(), &until, true); err != nil {
			return err
		}
		if err := rt.Client.UnbanChatMember(ctx, rt.ChatID(), rt.ActorID(), true); err != nil {
			return err
		}
		return s.sendBlocklistActionMessage(ctx, rt, "Kicked", reason)
	case "mute":
		if err := rt.Client.RestrictChatMember(ctx, rt.ChatID(), rt.ActorID(), telegram.RestrictPermissions{CanSendMessages: false}, nil); err != nil {
			return err
		}
		return s.sendBlocklistActionMessage(ctx, rt, "Muted", reason)
	case "tban":
		if action.DurationSeconds <= 0 {
			return fmt.Errorf("temporary blocklist ban duration is not configured")
		}
		until := time.Now().Add(time.Duration(action.DurationSeconds) * time.Second)
		if err := rt.Client.BanChatMember(ctx, rt.ChatID(), rt.ActorID(), &until, true); err != nil {
			return err
		}
		return s.sendBlocklistTimedActionMessage(ctx, rt, "Temp-banned", action.DurationSeconds, reason)
	case "tmute":
		if action.DurationSeconds <= 0 {
			return fmt.Errorf("temporary blocklist mute duration is not configured")
		}
		until := time.Now().Add(time.Duration(action.DurationSeconds) * time.Second)
		if err := rt.Client.RestrictChatMember(ctx, rt.ChatID(), rt.ActorID(), telegram.RestrictPermissions{CanSendMessages: false}, &until); err != nil {
			return err
		}
		return s.sendBlocklistTimedActionMessage(ctx, rt, "Temp-muted", action.DurationSeconds, reason)
	default:
		return fmt.Errorf("unsupported blocklist action %q", action.Action)
	}
}

func (s *Service) sendBlocklistActionMessage(ctx context.Context, rt *runtime.Context, verb string, reason string) error {
	name := strconv.FormatInt(rt.ActorID(), 10)
	if rt.Message != nil && rt.Message.From != nil {
		name = serviceutil.DisplayName(*rt.Message.From)
	}
	text := fmt.Sprintf("%s %s.", verb, name)
	if strings.TrimSpace(reason) != "" {
		text += " Reason: " + reason
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	_ = serviceutil.SendLog(ctx, rt, fmt.Sprintf("blocklist: actor=%d action=%s reason=%s", rt.ActorID(), strings.ToLower(verb), reason))
	return err
}

func (s *Service) sendBlocklistTimedActionMessage(ctx context.Context, rt *runtime.Context, verb string, durationSeconds int, reason string) error {
	name := strconv.FormatInt(rt.ActorID(), 10)
	if rt.Message != nil && rt.Message.From != nil {
		name = serviceutil.DisplayName(*rt.Message.From)
	}
	text := fmt.Sprintf("%s %s for %s.", verb, name, humanizeFlexibleDuration(time.Duration(durationSeconds)*time.Second))
	if strings.TrimSpace(reason) != "" {
		text += " Reason: " + reason
	}
	_, err := rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	_ = serviceutil.SendLog(ctx, rt, fmt.Sprintf("blocklist: actor=%d action=%s duration=%d reason=%s", rt.ActorID(), strings.ToLower(verb), durationSeconds, reason))
	return err
}

func (s *Service) warnBlocklistUser(ctx context.Context, rt *runtime.Context, reason string, pattern string) error {
	count, err := rt.Store.IncrementWarnings(ctx, rt.Bot.ID, rt.ChatID(), rt.ActorID(), reason)
	if err != nil {
		return err
	}
	name := strconv.FormatInt(rt.ActorID(), 10)
	if rt.Message != nil && rt.Message.From != nil {
		name = serviceutil.DisplayName(*rt.Message.From)
	}
	text := fmt.Sprintf("%s now has %d warning(s).", name, count)
	if strings.TrimSpace(reason) != "" {
		text += " Reason: " + reason
	}
	if rt.RuntimeBundle.Moderation.WarnLimit > 0 && count >= rt.RuntimeBundle.Moderation.WarnLimit {
		switch strings.ToLower(rt.RuntimeBundle.Moderation.WarnMode) {
		case "ban":
			if err := rt.Client.BanChatMember(ctx, rt.ChatID(), rt.ActorID(), nil, true); err != nil {
				return err
			}
			text = fmt.Sprintf("%s hit %d warnings and was banned.", name, count)
		case "kick":
			until := time.Now().Add(30 * time.Second)
			if err := rt.Client.BanChatMember(ctx, rt.ChatID(), rt.ActorID(), &until, true); err != nil {
				return err
			}
			if err := rt.Client.UnbanChatMember(ctx, rt.ChatID(), rt.ActorID(), true); err != nil {
				return err
			}
			text = fmt.Sprintf("%s hit %d warnings and was kicked.", name, count)
		default:
			if err := rt.Client.RestrictChatMember(ctx, rt.ChatID(), rt.ActorID(), telegram.RestrictPermissions{CanSendMessages: false}, nil); err != nil {
				return err
			}
			text = fmt.Sprintf("%s hit %d warnings and was muted.", name, count)
		}
	}
	_, err = rt.Client.SendMessage(ctx, rt.ChatID(), text, rt.ReplyOptions(telegram.SendMessageOptions{}))
	_ = serviceutil.SendLog(ctx, rt, fmt.Sprintf("blocklist: actor=%d action=warn pattern=%s count=%d reason=%s", rt.ActorID(), pattern, count, reason))
	return err
}

func normalizeBlocklistText(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func matchWordPattern(text string, pattern string) bool {
	if text == "" || pattern == "" {
		return false
	}
	if !containsWordChars(pattern) {
		return strings.Contains(text, pattern)
	}
	re := regexp.MustCompile(`(^|[^\p{L}\p{N}_])` + regexp.QuoteMeta(pattern) + `($|[^\p{L}\p{N}_])`)
	return re.MatchString(text)
}

func matchExactPattern(text string, pattern string) bool {
	if strings.Contains(pattern, "?") || strings.Contains(pattern, "*") {
		re, err := regexp.Compile("^" + blocklistWildcardRegex(pattern) + "$")
		return err == nil && re.MatchString(text)
	}
	return text == pattern
}

func matchPrefixPattern(text string, pattern string) bool {
	if strings.Contains(pattern, "?") || strings.Contains(pattern, "*") {
		re, err := regexp.Compile("^" + blocklistWildcardRegex(pattern))
		return err == nil && re.MatchString(text)
	}
	return strings.HasPrefix(text, pattern)
}

func matchWildcardPattern(text string, pattern string) bool {
	if text == "" || pattern == "" {
		return false
	}
	re, err := regexp.Compile(blocklistWildcardRegex(pattern))
	return err == nil && re.MatchString(text)
}

func blocklistWildcardRegex(pattern string) string {
	pattern = normalizeBlocklistText(pattern)
	var builder strings.Builder
	for i := 0; i < len(pattern); i++ {
		switch {
		case i+1 < len(pattern) && pattern[i] == '*' && pattern[i+1] == '*':
			builder.WriteString(".*")
			i++
		case pattern[i] == '*':
			builder.WriteString(`\S*`)
		case pattern[i] == '?':
			builder.WriteString(`\S`)
		default:
			builder.WriteString(regexp.QuoteMeta(string(pattern[i])))
		}
	}
	return builder.String()
}

func matchForwardPattern(message *telegram.Message, pattern string) bool {
	if message == nil {
		return false
	}
	candidates := make([]string, 0, 2)
	if message.ForwardFromChat != nil && message.ForwardFromChat.Username != "" {
		candidates = append(candidates, strings.ToLower("@"+strings.TrimPrefix(message.ForwardFromChat.Username, "@")))
	}
	if message.SenderChat != nil && message.SenderChat.Username != "" {
		candidates = append(candidates, strings.ToLower("@"+strings.TrimPrefix(message.SenderChat.Username, "@")))
	}
	if originMap, ok := message.ForwardOrigin.(map[string]any); ok {
		if chatMap, ok := originMap["chat"].(map[string]any); ok {
			if username, ok := chatMap["username"].(string); ok && username != "" {
				candidates = append(candidates, strings.ToLower("@"+strings.TrimPrefix(username, "@")))
			}
		}
	}
	for _, candidate := range candidates {
		if matchWildcardPattern(candidate, pattern) || candidate == pattern {
			return true
		}
	}
	return false
}

func normalizeLookalikeText(value string) string {
	replacer := strings.NewReplacer(
		"а", "a", "е", "e", "о", "o", "р", "p", "с", "c", "у", "y", "х", "x", "в", "b", "і", "i", "ј", "j", "к", "k", "м", "m", "н", "h", "т", "t",
		"А", "a", "Е", "e", "О", "o", "Р", "p", "С", "c", "У", "y", "Х", "x", "В", "b", "І", "i", "Ј", "j", "К", "k", "М", "m", "Н", "h", "Т", "t",
		"ο", "o", "Ο", "o", "Β", "b", "Ь", "b",
	)
	return normalizeBlocklistText(replacer.Replace(value))
}

func containsWordChars(value string) bool {
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
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
