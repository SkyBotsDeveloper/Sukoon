package content

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"sukoon/bot-core/internal/domain"
	"sukoon/bot-core/internal/runtime"
	"sukoon/bot-core/internal/serviceutil"
	"sukoon/bot-core/internal/telegram"
)

func parseStoredContent(raw string) (string, string, error) {
	textLines := make([]string, 0)
	buttons := make([][]telegram.InlineKeyboardButton, 0)

	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		row, isRow, err := parseButtonRow(line)
		if err != nil {
			return "", "", err
		}
		if isRow {
			buttons = append(buttons, row)
			continue
		}
		textLines = append(textLines, line)
	}

	body, err := json.Marshal(buttons)
	if err != nil {
		return "", "", err
	}
	return strings.TrimSpace(strings.Join(textLines, "\n")), string(body), nil
}

func buttonsFromJSON(raw string) (*telegram.InlineKeyboardMarkup, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var buttons [][]telegram.InlineKeyboardButton
	if err := json.Unmarshal([]byte(raw), &buttons); err != nil {
		return nil, err
	}
	if len(buttons) == 0 {
		return nil, nil
	}
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: buttons}, nil
}

func renderStoredText(raw string, user telegram.User, chat telegram.Chat, rules string) string {
	return serviceutil.RenderStoredMessage(raw, user, chat, rules)
}

func splitNameAndBody(raw string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", fmt.Errorf("name is required")
	}
	for idx, r := range raw {
		if r == ' ' || r == '\n' || r == '\t' {
			name := strings.TrimSpace(raw[:idx])
			body := strings.TrimSpace(raw[idx+1:])
			if body == "" {
				return "", "", fmt.Errorf("content is required")
			}
			return name, body, nil
		}
	}
	return "", "", fmt.Errorf("content is required")
}

func splitTriggerAndBody(raw string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", fmt.Errorf("trigger is required")
	}
	if strings.HasPrefix(raw, "\"") {
		end := strings.Index(raw[1:], "\"")
		if end < 0 {
			return "", "", fmt.Errorf("quoted trigger is missing a closing quote")
		}
		trigger := strings.TrimSpace(raw[1 : end+1])
		body := strings.TrimSpace(raw[end+2:])
		if trigger == "" || body == "" {
			return "", "", fmt.Errorf("usage: /filter <trigger> <response>")
		}
		return trigger, body, nil
	}
	return splitNameAndBody(raw)
}

type filterDefinition struct {
	Trigger   string
	MatchMode string
	Body      string
}

func parseFilterDefinitions(raw string) ([]filterDefinition, error) {
	triggers, body, err := splitFilterTriggersAndBody(raw)
	if err != nil {
		return nil, err
	}
	definitions := make([]filterDefinition, 0, len(triggers))
	seen := map[string]struct{}{}
	for _, trigger := range triggers {
		trigger = strings.TrimSpace(trigger)
		if trigger == "" {
			continue
		}
		matchMode := "contains"
		lower := strings.ToLower(trigger)
		switch {
		case strings.HasPrefix(lower, "exact:"):
			matchMode = "exact"
			trigger = strings.TrimSpace(trigger[len("exact:"):])
		case strings.HasPrefix(lower, "prefix:"):
			matchMode = "prefix"
			trigger = strings.TrimSpace(trigger[len("prefix:"):])
		}
		if trigger == "" {
			continue
		}
		key := matchMode + ":" + strings.ToLower(trigger)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		definitions = append(definitions, filterDefinition{Trigger: trigger, MatchMode: matchMode, Body: body})
	}
	if len(definitions) == 0 {
		return nil, fmt.Errorf("trigger is required")
	}
	return definitions, nil
}

func splitFilterTriggersAndBody(raw string) ([]string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, "", fmt.Errorf("trigger is required")
	}
	if strings.HasPrefix(raw, "(") {
		end := findClosingParen(raw)
		if end < 0 {
			return nil, "", fmt.Errorf("multi-filter trigger list is missing a closing bracket")
		}
		body := strings.TrimSpace(raw[end+1:])
		if body == "" {
			return nil, "", fmt.Errorf("usage: /filter <trigger> <response>")
		}
		triggers, err := splitCommaTriggers(raw[1:end])
		if err != nil {
			return nil, "", err
		}
		return triggers, body, nil
	}
	trigger, body, err := splitTriggerAndBody(raw)
	if err != nil {
		return nil, "", err
	}
	return []string{trigger}, body, nil
}

func parseFilterTriggersOnly(raw string) ([]filterDefinition, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("trigger is required")
	}
	var triggers []string
	var err error
	if strings.HasPrefix(raw, "(") {
		end := findClosingParen(raw)
		if end < 0 || strings.TrimSpace(raw[end+1:]) != "" {
			return nil, fmt.Errorf("usage: /filter <trigger> <response>")
		}
		triggers, err = splitCommaTriggers(raw[1:end])
		if err != nil {
			return nil, err
		}
	} else {
		trimmed := strings.TrimSpace(raw)
		trigger := ""
		switch {
		case strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"") && len(trimmed) >= 2:
			trigger = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
		default:
			trigger = strings.TrimSpace(trimmed)
			if strings.ContainsAny(trigger, " \n\t") {
				return nil, fmt.Errorf("usage: /filter <trigger> <response>")
			}
		}
		if trigger == "" {
			return nil, fmt.Errorf("usage: /filter <trigger> <response>")
		}
		triggers = []string{trigger}
	}

	definitions := make([]filterDefinition, 0, len(triggers))
	seen := map[string]struct{}{}
	for _, trigger := range triggers {
		trigger = strings.TrimSpace(trigger)
		if trigger == "" {
			continue
		}
		matchMode := "contains"
		lower := strings.ToLower(trigger)
		switch {
		case strings.HasPrefix(lower, "exact:"):
			matchMode = "exact"
			trigger = strings.TrimSpace(trigger[len("exact:"):])
		case strings.HasPrefix(lower, "prefix:"):
			matchMode = "prefix"
			trigger = strings.TrimSpace(trigger[len("prefix:"):])
		}
		if trigger == "" {
			continue
		}
		key := matchMode + ":" + strings.ToLower(trigger)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		definitions = append(definitions, filterDefinition{Trigger: trigger, MatchMode: matchMode})
	}
	if len(definitions) == 0 {
		return nil, fmt.Errorf("trigger is required")
	}
	return definitions, nil
}

func findClosingParen(raw string) int {
	inQuote := false
	for idx, r := range raw {
		switch r {
		case '"':
			inQuote = !inQuote
		case ')':
			if !inQuote {
				return idx
			}
		}
	}
	return -1
}

func splitCommaTriggers(raw string) ([]string, error) {
	var triggers []string
	var current strings.Builder
	inQuote := false
	for _, r := range raw {
		switch r {
		case '"':
			inQuote = !inQuote
		case ',':
			if !inQuote {
				if item := strings.TrimSpace(current.String()); item != "" {
					triggers = append(triggers, strings.Trim(item, `"`))
				}
				current.Reset()
				continue
			}
		}
		current.WriteRune(r)
	}
	if inQuote {
		return nil, fmt.Errorf("quoted trigger is missing a closing quote")
	}
	if item := strings.TrimSpace(current.String()); item != "" {
		triggers = append(triggers, strings.Trim(item, `"`))
	}
	return triggers, nil
}

type filterRenderResult struct {
	Text                  string
	ReplyMarkup           *telegram.InlineKeyboardMarkup
	DisableWebPagePreview bool
	EnableWebPagePreview  bool
	ShowPreviewAboveText  bool
	DisableNotification   bool
	ProtectContent        bool
	HasMediaSpoiler       bool
	MediaType             string
	MediaFileID           string
}

func renderFilterResponse(filter domain.FilterRule, user telegram.User, chat telegram.Chat, rules string, rt *runtime.Context, noFormat bool, force bool) (filterRenderResult, bool, error) {
	if noFormat {
		replyMarkup, err := buttonsFromJSON(filter.ButtonsJSON)
		if err != nil {
			return filterRenderResult{}, false, err
		}
		text := filter.ResponseText
		if strings.TrimSpace(text) == "" && filter.ResponseMediaFileID != "" {
			text = "This filter replies with saved media."
		}
		return filterRenderResult{Text: text, ReplyMarkup: replyMarkup}, true, nil
	}

	segments := strings.Split(filter.ResponseText, "%%%")
	permitted := make([]string, 0, len(segments))
	actorIsAdmin := rt.ActorPermissions.IsOwner || rt.ActorPermissions.IsSudo || rt.ActorPermissions.IsChatAdmin || rt.ActorPermissions.IsSilentMod
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		adminOnly := strings.Contains(segment, "{admin}")
		userOnly := strings.Contains(segment, "{user}")
		switch {
		case adminOnly && !actorIsAdmin:
			continue
		case userOnly && actorIsAdmin && !force:
			continue
		}
		permitted = append(permitted, segment)
	}
	if len(permitted) == 0 {
		return filterRenderResult{}, false, nil
	}
	chosen := permitted[0]
	if len(permitted) > 1 {
		chosen = permitted[rand.Intn(len(permitted))]
	}

	renderUser := user
	if strings.Contains(chosen, "{replytag}") && rt.Message != nil && rt.Message.ReplyToMessage != nil && rt.Message.ReplyToMessage.From != nil {
		renderUser = *rt.Message.ReplyToMessage.From
	}
	rendered, err := renderStoredPayload(chosen, filter.ButtonsJSON, renderUser, chat, rules, rt.Bot.Username, rt.ChatID())
	if err != nil {
		return filterRenderResult{}, false, err
	}
	return filterRenderResult{
		Text:                  rendered.Text,
		ReplyMarkup:           rendered.ReplyMarkup,
		DisableWebPagePreview: rendered.DisableWebPagePreview,
		EnableWebPagePreview:  rendered.EnableWebPagePreview,
		ShowPreviewAboveText:  rendered.ShowPreviewAboveText,
		DisableNotification:   rendered.DisableNotification,
		ProtectContent:        rendered.ProtectContent,
		HasMediaSpoiler:       rendered.HasMediaSpoiler,
		MediaType:             filter.ResponseMediaType,
		MediaFileID:           filter.ResponseMediaFileID,
	}, true, nil
}

func stripFilterControlTokens(raw string) string {
	replacer := strings.NewReplacer(
		"{admin}", "",
		"{user}", "",
		"{force}", "",
		"{replytag}", "",
		"{protect}", "",
		"{nonotif}", "",
		"{preview}", "",
		"{preview:top}", "",
		"{rules}", "",
		"{rules:same}", "",
		"{mediaspoiler}", "",
	)
	return strings.TrimSpace(replacer.Replace(raw))
}

type storedPayloadResult struct {
	Text                  string
	ReplyMarkup           *telegram.InlineKeyboardMarkup
	DisableWebPagePreview bool
	EnableWebPagePreview  bool
	ShowPreviewAboveText  bool
	DisableNotification   bool
	ProtectContent        bool
	HasMediaSpoiler       bool
}

func renderStoredPayload(raw string, buttonsJSON string, user telegram.User, chat telegram.Chat, rules string, botUsername string, rulesChatID int64) (storedPayloadResult, error) {
	replyMarkup, err := buttonsFromJSON(buttonsJSON)
	if err != nil {
		return storedPayloadResult{}, err
	}

	hasRulesRow := strings.Contains(raw, "{rules}")
	hasRulesSame := strings.Contains(raw, "{rules:same}")
	enablePreview := strings.Contains(raw, "{preview}") || strings.Contains(raw, "{preview:top}")
	showPreviewAbove := strings.Contains(raw, "{preview:top}")

	cleaned := stripFilterControlTokens(raw)
	if hasRulesRow || hasRulesSame {
		replyMarkup = appendRulesButton(replyMarkup, botUsername, rulesChatID, strings.TrimSpace(rules) != "", hasRulesRow, hasRulesSame)
	}

	return storedPayloadResult{
		Text:                  renderStoredText(cleaned, user, chat, rules),
		ReplyMarkup:           replyMarkup,
		DisableWebPagePreview: !enablePreview,
		EnableWebPagePreview:  enablePreview,
		ShowPreviewAboveText:  showPreviewAbove,
		DisableNotification:   strings.Contains(raw, "{nonotif}"),
		ProtectContent:        strings.Contains(raw, "{protect}"),
		HasMediaSpoiler:       strings.Contains(raw, "{mediaspoiler}"),
	}, nil
}

func appendRulesButton(markup *telegram.InlineKeyboardMarkup, botUsername string, rulesChatID int64, hasRules bool, newRow bool, sameRow bool) *telegram.InlineKeyboardMarkup {
	if !hasRules || strings.TrimSpace(botUsername) == "" || rulesChatID == 0 {
		return markup
	}

	rows := make([][]telegram.InlineKeyboardButton, 0)
	if markup != nil {
		rows = make([][]telegram.InlineKeyboardButton, 0, len(markup.InlineKeyboard)+1)
		for _, row := range markup.InlineKeyboard {
			rows = append(rows, append([]telegram.InlineKeyboardButton(nil), row...))
		}
	}

	button := telegram.InlineKeyboardButton{
		Text: "Rules",
		URL:  serviceutil.BotDeepLink(botUsername, fmt.Sprintf("rules_%d", rulesChatID)),
	}
	if sameRow && len(rows) > 0 {
		rows[len(rows)-1] = append(rows[len(rows)-1], button)
	} else if newRow || len(rows) == 0 {
		rows = append(rows, []telegram.InlineKeyboardButton{button})
	} else {
		rows[len(rows)-1] = append(rows[len(rows)-1], button)
	}
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func mediaFromMessage(message *telegram.Message) (string, string, bool) {
	if message == nil {
		return "", "", false
	}
	switch {
	case len(message.Photo) > 0:
		return "photo", message.Photo[len(message.Photo)-1].FileID, true
	case message.Animation != nil && message.Animation.FileID != "":
		return "animation", message.Animation.FileID, true
	case message.Video != nil && message.Video.FileID != "":
		return "video", message.Video.FileID, true
	case message.Document != nil && message.Document.FileID != "":
		return "document", message.Document.FileID, true
	case message.Audio != nil && message.Audio.FileID != "":
		return "audio", message.Audio.FileID, true
	case message.Voice != nil && message.Voice.FileID != "":
		return "voice", message.Voice.FileID, true
	case message.VideoNote != nil && message.VideoNote.FileID != "":
		return "videonote", message.VideoNote.FileID, true
	case message.Sticker != nil && message.Sticker.FileID != "":
		return "sticker", message.Sticker.FileID, true
	default:
		return "", "", false
	}
}

func parseFilterReplyPayload(message *telegram.Message) (string, string, string, string, error) {
	if message == nil {
		return "", "", "", "", fmt.Errorf("usage: /filter <trigger> <response>")
	}
	body := strings.TrimSpace(message.Caption)
	if body == "" {
		body = strings.TrimSpace(message.Text)
	}
	text, buttonsJSON, err := parseStoredContent(body)
	if err != nil {
		return "", "", "", "", err
	}
	mediaType, mediaFileID, hasMedia := mediaFromMessage(message)
	if hasMedia {
		return text, buttonsJSON, mediaType, mediaFileID, nil
	}
	if strings.TrimSpace(text) == "" {
		return "", "", "", "", fmt.Errorf("usage: /filter <trigger> <response>")
	}
	return text, buttonsJSON, "", "", nil
}

func parseButtonRow(line string) ([]telegram.InlineKeyboardButton, bool, error) {
	line = strings.TrimSpace(line)
	if line == "" || !strings.HasPrefix(line, "[") {
		return nil, false, nil
	}

	rest := line
	row := make([]telegram.InlineKeyboardButton, 0, 2)
	for strings.TrimSpace(rest) != "" {
		rest = strings.TrimLeft(rest, " ")
		if !strings.HasPrefix(rest, "[") {
			return nil, false, nil
		}
		closeLabel := strings.Index(rest, "]")
		if closeLabel <= 1 {
			return nil, false, fmt.Errorf("invalid button label")
		}
		label := rest[1:closeLabel]
		rest = rest[closeLabel+1:]
		if !strings.HasPrefix(rest, "(") {
			return nil, false, nil
		}
		closeAction := strings.Index(rest, ")")
		if closeAction <= 1 {
			return nil, false, fmt.Errorf("invalid button action")
		}
		action := rest[1:closeAction]
		rest = strings.TrimLeft(rest[closeAction+1:], " ")

		parts := strings.SplitN(action, ":", 2)
		if len(parts) != 2 {
			return nil, false, fmt.Errorf("button action must include a value")
		}

		button := telegram.InlineKeyboardButton{Text: label}
		switch strings.ToLower(parts[0]) {
		case "buttonurl":
			button.URL = strings.TrimSpace(parts[1])
		case "button":
			button.CallbackData = strings.TrimSpace(parts[1])
		default:
			return nil, false, fmt.Errorf("unsupported button type %s", parts[0])
		}
		row = append(row, button)
	}
	return row, len(row) > 0, nil
}
