package content

import (
	"encoding/json"
	"fmt"
	"strings"

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
