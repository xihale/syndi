package routes

import (
	"bytes"
	"encoding/json"
	"strconv"
)

// lcFlexString accepts a JSON string, number or bool and stores its raw text.
// LeetCode GraphQL occasionally returns numeric fields as either JSON numbers
// or strings; this keeps unmarshalling tolerant of both shapes.
type lcFlexString string

// UnmarshalJSON implements json.Unmarshaler tolerantly.
func (s *lcFlexString) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*s = ""
		return nil
	}
	if data[0] == '"' {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*s = lcFlexString(v)
		return nil
	}
	*s = lcFlexString(data)
	return nil
}

// String returns the stored value as a plain string.
func (s lcFlexString) String() string { return string(s) }

// lcFlexInt64 accepts a JSON number or a quoted number string.
type lcFlexInt64 int64

// UnmarshalJSON implements json.Unmarshaler tolerantly.
func (n *lcFlexInt64) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*n = 0
		return nil
	}
	if data[0] == '"' {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		parsed, err := strconv.ParseInt(trimLCSpaces(v), 10, 64)
		if err != nil {
			if f, ferr := strconv.ParseFloat(trimLCSpaces(v), 64); ferr == nil {
				*n = lcFlexInt64(int64(f))
				return nil
			}
			return err
		}
		*n = lcFlexInt64(parsed)
		return nil
	}
	parsed, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		var f float64
		if ferr := json.Unmarshal(data, &f); ferr == nil {
			*n = lcFlexInt64(int64(f))
			return nil
		}
		return err
	}
	*n = lcFlexInt64(parsed)
	return nil
}

// Int64 returns the stored value.
func (n lcFlexInt64) Int64() int64 { return int64(n) }

func trimLCSpaces(v string) string {
	start, end := 0, len(v)
	for start < end && (v[start] == ' ' || v[start] == '\t') {
		start++
	}
	for end > start && (v[end-1] == ' ' || v[end-1] == '\t') {
		end--
	}
	return v[start:end]
}
