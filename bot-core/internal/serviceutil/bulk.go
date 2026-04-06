package serviceutil

import "strings"

func SplitBulkItems(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	separator := ""
	switch {
	case strings.Contains(raw, "|"):
		separator = "|"
	case strings.Contains(raw, "\n"):
		separator = "\n"
	case strings.Contains(raw, ","):
		separator = ","
	default:
		return []string{raw}
	}

	parts := strings.Split(raw, separator)
	items := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		items = append(items, item)
	}
	return items
}
