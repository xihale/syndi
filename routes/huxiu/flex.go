package routes

import (
	"encoding/json"
	"strconv"
	"strings"
)

// flexInt64 accepts a JSON number or a numeric string, defaulting to 0.
type flexInt64 int64

// UnmarshalJSON implements tolerant numeric decoding.
func (f *flexInt64) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		*f = flexInt64(n)
		return nil
	}
	if fl, err := strconv.ParseFloat(s, 64); err == nil {
		*f = flexInt64(fl)
		return nil
	}
	*f = 0
	return nil
}

// Int returns the value as int64.
func (f flexInt64) Int() int64 { return int64(f) }

// flexString accepts a JSON string, number or bool, defaulting to "".
type flexString string

// UnmarshalJSON implements tolerant string decoding.
func (f *flexString) UnmarshalJSON(b []byte) error {
	raw := strings.TrimSpace(string(b))
	if raw == "" || raw == "null" {
		*f = ""
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*f = flexString(s)
		return nil
	}
	// Numbers and booleans arrive as bare tokens; use their literal text.
	*f = flexString(raw)
	return nil
}

// String returns the value as string.
func (f flexString) String() string { return string(f) }

// flexStringSlice accepts an array of strings, a single string, or null.
type flexStringSlice []string

// UnmarshalJSON implements tolerant slice decoding.
func (f *flexStringSlice) UnmarshalJSON(b []byte) error {
	raw := strings.TrimSpace(string(b))
	if raw == "" || raw == "null" {
		*f = nil
		return nil
	}
	var list []flexString
	if err := json.Unmarshal(b, &list); err == nil {
		out := make(flexStringSlice, 0, len(list))
		for _, v := range list {
			if s := v.String(); s != "" {
				out = append(out, s)
			}
		}
		*f = out
		return nil
	}
	var single flexString
	if err := json.Unmarshal(b, &single); err == nil && single.String() != "" {
		*f = flexStringSlice{single.String()}
		return nil
	}
	*f = nil
	return nil
}
