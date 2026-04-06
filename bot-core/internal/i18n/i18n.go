package i18n

import "fmt"

var messages = map[string]map[string]string{
	"en": {
		"language.current": "Current language: %s",
		"language.updated": "Language updated to %s.",
		"privacy.info":     "Use /mydata to export your stored data and /forgetme confirm to delete supported self-service data.",
		"privacy.deleted":  "Your supported self-service data has been deleted.",
		"privacy.export":   "Your data export:\n%s",
	},
	"hi": {
		"language.current": "Current language: %s",
		"language.updated": "%s bhasha set ho gayi hai.",
		"privacy.info":     "Apna stored data export karne ke liye /mydata aur supported self-service data delete karne ke liye /forgetme confirm use karein.",
		"privacy.deleted":  "Aapka supported self-service data delete kar diya gaya hai.",
		"privacy.export":   "Aapka data export:\n%s",
	},
}

func SupportedLanguages() []string {
	return []string{"en", "hi"}
}

func IsSupported(language string) bool {
	_, ok := messages[language]
	return ok
}

func T(language string, key string, args ...any) string {
	langSet, ok := messages[language]
	if !ok {
		langSet = messages["en"]
	}
	template, ok := langSet[key]
	if !ok {
		template = messages["en"][key]
	}
	if len(args) == 0 {
		return template
	}
	return fmt.Sprintf(template, args...)
}
