package routes

import (
	"encoding/json"
	"strconv"
	"strings"
)

// dgFlexInt64 accepts a JSON number or a numeric string, defaulting to 0.
type dgFlexInt64 int64

// UnmarshalJSON implements tolerant numeric decoding.
func (f *dgFlexInt64) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		*f = dgFlexInt64(n)
		return nil
	}
	if fl, err := strconv.ParseFloat(s, 64); err == nil {
		*f = dgFlexInt64(fl)
		return nil
	}
	*f = 0
	return nil
}

// Int returns the value as int64.
func (f dgFlexInt64) Int() int64 { return int64(f) }

// dgFlexString accepts a JSON string, number or bool, defaulting to "".
type dgFlexString string

// UnmarshalJSON implements tolerant string decoding.
func (f *dgFlexString) UnmarshalJSON(b []byte) error {
	raw := strings.TrimSpace(string(b))
	if raw == "" || raw == "null" {
		*f = ""
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*f = dgFlexString(s)
		return nil
	}
	*f = dgFlexString(raw)
	return nil
}

// String returns the value as string.
func (f dgFlexString) String() string { return string(f) }

// dgFlexStringSlice accepts an array of strings, a single string, or null.
type dgFlexStringSlice []string

// UnmarshalJSON implements tolerant slice decoding.
func (f *dgFlexStringSlice) UnmarshalJSON(b []byte) error {
	raw := strings.TrimSpace(string(b))
	if raw == "" || raw == "null" {
		*f = nil
		return nil
	}
	var list []dgFlexString
	if err := json.Unmarshal(b, &list); err == nil {
		out := make(dgFlexStringSlice, 0, len(list))
		for _, v := range list {
			if s := v.String(); s != "" {
				out = append(out, s)
			}
		}
		*f = out
		return nil
	}
	var single dgFlexString
	if err := json.Unmarshal(b, &single); err == nil && single.String() != "" {
		*f = dgFlexStringSlice{single.String()}
		return nil
	}
	*f = nil
	return nil
}
