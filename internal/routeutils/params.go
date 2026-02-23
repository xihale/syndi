package routeutils

import (
	"strconv"
	"strings"
)

// ParsePositiveInt parses a positive integer with default and max bounds.
// Invalid, empty, or non-positive values fall back to defaultValue.
func ParsePositiveInt(raw string, defaultValue, maxValue int) int {
	if defaultValue <= 0 {
		defaultValue = 1
	}
	if maxValue <= 0 {
		maxValue = defaultValue
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || parsed <= 0 {
		return defaultValue
	}

	if parsed > maxValue {
		return maxValue
	}

	return parsed
}

// ParseOptionalPositiveInt parses a positive integer and returns nil when unset or invalid.
func ParseOptionalPositiveInt(raw string) *int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return nil
	}

	return &parsed
}

// ParseBool parses common true/false strings with a default fallback.
func ParseBool(raw string, defaultValue bool) bool {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return defaultValue
	}

	switch raw {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return defaultValue
	}
}

// ParseEnum normalizes and validates a value against allowed options.
// Matching is case-insensitive and returns the canonical allowed value.
func ParseEnum(raw, defaultValue string, allowed ...string) string {
	raw = strings.TrimSpace(raw)
	defaultValue = strings.TrimSpace(defaultValue)
	if len(allowed) == 0 {
		return defaultValue
	}

	for _, candidate := range allowed {
		if strings.EqualFold(raw, candidate) {
			return candidate
		}
	}

	for _, candidate := range allowed {
		if strings.EqualFold(defaultValue, candidate) {
			return candidate
		}
	}

	return allowed[0]
}
