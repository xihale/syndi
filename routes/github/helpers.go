package routes

import "strconv"

func parsePositiveLimit(raw string, defaultValue, maxValue int) int {
	if maxValue <= 0 {
		maxValue = defaultValue
	}
	if defaultValue <= 0 {
		defaultValue = 1
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return defaultValue
	}
	if parsed > maxValue {
		return maxValue
	}
	return parsed
}

func parseBoolDefault(raw string, defaultValue bool) bool {
	if raw == "" {
		return defaultValue
	}

	switch raw {
	case "1", "true", "TRUE", "True", "yes", "YES", "Yes":
		return true
	case "0", "false", "FALSE", "False", "no", "NO", "No":
		return false
	default:
		return defaultValue
	}
}
