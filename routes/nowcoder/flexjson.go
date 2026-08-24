package routes

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// flexString accepts a JSON string, number or bool and stores its raw text.
// Some upstream APIs return numeric ids as strings in some responses and as
// numbers in others; this keeps unmarshalling tolerant of both shapes.
type flexString string

// UnmarshalJSON implements json.Unmarshaler tolerantly.
func (s *flexString) UnmarshalJSON(data []byte) error {
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
		*s = flexString(v)
		return nil
	}
	*s = flexString(data)
	return nil
}

// String returns the stored value as a plain string.
func (s flexString) String() string { return string(s) }

// flexInt64 accepts a JSON number or a quoted number string and stores an int64.
type flexInt64 int64

// UnmarshalJSON implements json.Unmarshaler tolerantly.
func (n *flexInt64) UnmarshalJSON(data []byte) error {
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
		v = strings.TrimSpace(v)
		if v == "" {
			*n = 0
			return nil
		}
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			if f, ferr := strconv.ParseFloat(v, 64); ferr == nil {
				*n = flexInt64(int64(f))
				return nil
			}
			return err
		}
		*n = flexInt64(parsed)
		return nil
	}
	parsed, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		var f float64
		if ferr := json.Unmarshal(data, &f); ferr == nil {
			*n = flexInt64(int64(f))
			return nil
		}
		return err
	}
	*n = flexInt64(parsed)
	return nil
}

// Int64 returns the stored value.
func (n flexInt64) Int64() int64 { return int64(n) }
