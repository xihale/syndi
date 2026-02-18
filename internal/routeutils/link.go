package routeutils

import (
	"fmt"
	"net/url"
	"strings"
)

// AddQueryParam adds or replaces a query parameter
func AddQueryParam(u, key, value string) (string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	query := parsed.Query()
	query.Set(key, value)
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}

// RemoveQueryParam removes a query parameter
func RemoveQueryParam(u, key string) (string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	query := parsed.Query()
	query.Del(key)
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}

// StripQuery removes all query parameters
func StripQuery(u string) string {
	if idx := strings.Index(u, "?"); idx != -1 {
		return u[:idx]
	}
	// Also handle fragments
	if idx := strings.Index(u, "#"); idx != -1 {
		return u[:idx]
	}
	return u
}

// EnsureHTTPS ensures URL uses https scheme
func EnsureHTTPS(u string) (string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme != "https" {
		parsed.Scheme = "https"
	}

	return parsed.String(), nil
}

// AddFragment adds or replaces a URL fragment
func AddFragment(u, fragment string) (string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	parsed.Fragment = strings.TrimPrefix(fragment, "#")
	return parsed.String(), nil
}

// RemoveFragment removes the fragment from a URL
func RemoveFragment(u string) (string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	parsed.Fragment = ""
	return parsed.String(), nil
}

// GetQueryParam extracts a query parameter value from a URL
func GetQueryParam(u, key string) (string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	return parsed.Query().Get(key), nil
}

// HasQueryParam checks if a URL has a specific query parameter
func HasQueryParam(u, key string) (bool, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return false, fmt.Errorf("invalid URL: %w", err)
	}

	return parsed.Query().Has(key), nil
}

// GetHostname extracts the hostname from a URL
func GetHostname(u string) (string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	return parsed.Hostname(), nil
}

// GetPath extracts the path from a URL
func GetPath(u string) (string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	return parsed.Path, nil
}

// ParseQuery parses URL query parameters into a map
func ParseQuery(u string) (map[string]string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	result := make(map[string]string)
	for key, values := range parsed.Query() {
		if len(values) > 0 {
			result[key] = values[0]
		}
	}

	return result, nil
}

// BuildQuery builds a query string from a map
func BuildQuery(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}

	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}

	return values.Encode()
}

// MergeQueryParams merges query parameters into a URL
func MergeQueryParams(u string, params map[string]string) (string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	query := parsed.Query()
	for key, value := range params {
		query.Set(key, value)
	}
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}

// IsHTTPS checks if a URL uses HTTPS scheme
func IsHTTPS(u string) (bool, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return false, fmt.Errorf("invalid URL: %w", err)
	}

	return parsed.Scheme == "https", nil
}

// IsAbsoluteURL checks if a URL is absolute
func IsAbsoluteURL(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}

// IsProtocolRelativeURL checks if a URL is protocol-relative (//example.com)
func IsProtocolRelativeURL(u string) bool {
	return strings.HasPrefix(u, "//")
}

// GetDomain extracts the domain (including subdomain) from a URL
func GetDomain(u string) (string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	return parsed.Host, nil
}

// CleanURL removes tracking parameters and common clutter from URLs
func CleanURL(u string) (string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Remove fragment
	parsed.Fragment = ""

	// Remove common tracking parameters
	trackingParams := []string{
		"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content",
		"fbclid", "gclid", "msclkid",
		"_ga", "_gid",
		"ref", "source",
	}

	query := parsed.Query()
	for _, param := range trackingParams {
		query.Del(param)
	}
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}

// EncodeURLSegment encodes a URL path segment
func EncodeURLSegment(segment string) string {
	return url.QueryEscape(segment)
}

// DecodeURLSegment decodes a URL path segment
func DecodeURLSegment(segment string) (string, error) {
	return url.QueryUnescape(segment)
}
