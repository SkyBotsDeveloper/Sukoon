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
	Text                string
	ReplyMarkup         *telegram.InlineKeyboardMarkup
	DisableNotification bool
	ProtectContent      bool
}

func renderFilterResponse(filter domain.FilterRule, user telegram.User, chat telegram.Chat, rules string, rt *runtime.Context, noFormat bool, force bool) (filterRenderResult, bool, error) {
	replyMarkup, err := buttonsFromJSON(filter.ButtonsJSON)
	if err != nil {
		return filterRenderResult{}, false, err
	}
	if noFormat {
		return filterRenderResult{Text: filter.ResponseText, ReplyMarkup: replyMarkup}, true, nil
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
	result := filterRenderResult{
		DisableNotification: strings.Contains(chosen, "{nonotif}"),
		ProtectContent:      strings.Contains(chosen, "{protect}"),
		ReplyMarkup:         replyMarkup,
	}
	cleaned := stripFilterControlTokens(chosen)
	result.Text = renderStoredText(cleaned, renderUser, chat, rules)
	return result, true, nil
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
	)
	return strings.TrimSpace(replacer.Replace(raw))
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
