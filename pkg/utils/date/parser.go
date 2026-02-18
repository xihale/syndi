package date

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParseDate parses various date formats (simplified version)
// Full implementation would support natural language like "昨天", "周一", etc.
func ParseDate(input string) (time.Time, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return time.Time{}, nil
	}

	// Try common formats
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		time.RFC1123,
		time.RFC1123Z,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z07:00",
		"2006/01/02 15:04:05",
		"2006-01-02",
		"02 Jan 2006",
		"Jan 2, 2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, input); err == nil {
			return t, nil
		}
	}

	// Handle relative dates (simple version)
	return parseRelative(input)
}

func parseRelative(input string) (time.Time, error) {
	lower := strings.ToLower(input)

	// "now" or "just now"
	if lower == "now" || lower == "just now" {
		return time.Now(), nil
	}

	// Chinese relative dates
	if strings.Contains(input, "昨天") {
		return time.Now().AddDate(0, 0, -1), nil
	}
	if strings.Contains(input, "前天") {
		return time.Now().AddDate(0, 0, -2), nil
	}

	// English relative dates
	re := regexp.MustCompile(`(\d+)\s*(minute|hour|day|week|month|year)s?`)
	if matches := re.FindStringSubmatch(lower); len(matches) == 3 {
		num, _ := strconv.Atoi(matches[1])
		unit := matches[2]

		now := time.Now()
		switch unit {
		case "minute":
			return now.Add(time.Duration(-num) * time.Minute), nil
		case "hour":
			return now.Add(time.Duration(-num) * time.Hour), nil
		case "day":
			return now.AddDate(0, 0, -num), nil
		case "week":
			return now.AddDate(0, 0, -num*7), nil
		case "month":
			return now.AddDate(0, -num, 0), nil
		case "year":
			return now.AddDate(-num, 0, 0), nil
		}
	}

	return time.Time{}, nil
}
