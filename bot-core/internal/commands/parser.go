package commands

import "strings"

type Parsed struct {
	Name    string
	Args    []string
	RawArgs string
}

func Parse(text string, botUsername string) (Parsed, bool) {
	text = strings.TrimSpace(text)
	if text == "" || text[0] != '/' {
		return Parsed{}, false
	}

	fields := strings.Fields(text)
	if len(fields) == 0 {
		return Parsed{}, false
	}

	commandToken := strings.TrimPrefix(fields[0], "/")
	if commandToken == "" {
		return Parsed{}, false
	}

	name := strings.ToLower(commandToken)
	if idx := strings.IndexByte(name, '@'); idx >= 0 {
		target := strings.TrimPrefix(name[idx:], "@")
		if botUsername == "" || target != strings.ToLower(strings.TrimPrefix(botUsername, "@")) {
			return Parsed{}, false
		}
		name = name[:idx]
	}

	rawArgs := ""
	if len(text) > len(fields[0]) {
		rawArgs = strings.TrimSpace(text[len(fields[0]):])
	}

	return Parsed{
		Name:    name,
		Args:    fields[1:],
		RawArgs: rawArgs,
	}, true
}
