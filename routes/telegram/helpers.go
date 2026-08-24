package routes

import (
	"strings"

	"html"
)

// truncateText shortens plain text for use as a title.
func truncateText(s string, max int) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

// extractPlainText strips simple HTML tags from a fragment for titles.
func extractPlainText(s string) string {
	var sb strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			sb.WriteRune(r)
		}
	}
	text := strings.Join(strings.Fields(sb.String()), " ")
	return html.UnescapeString(text)
}
