package routes

import (
	"bytes"
	"encoding/json"
	"strconv"
)

// jkFlexString accepts a JSON string, number or bool and stores its raw text.
// Upstream embedded JSON sometimes switches a numeric field between number
// and string across releases; this keeps unmarshalling tolerant.
type jkFlexString string

// UnmarshalJSON implements json.Unmarshaler tolerantly.
func (s *jkFlexString) UnmarshalJSON(data []byte) error {
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
		*s = jkFlexString(v)
		return nil
	}
	*s = jkFlexString(data)
	return nil
}

// String returns the stored value as a plain string.
func (s jkFlexString) String() string { return string(s) }

// jkFlexInt64 accepts a JSON number or a quoted number string.
type jkFlexInt64 int64

// UnmarshalJSON implements json.Unmarshaler tolerantly.
func (n *jkFlexInt64) UnmarshalJSON(data []byte) error {
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
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			if f, ferr := strconv.ParseFloat(v, 64); ferr == nil {
				*n = jkFlexInt64(int64(f))
				return nil
			}
			return err
		}
		*n = jkFlexInt64(parsed)
		return nil
	}
	parsed, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		var f float64
		if ferr := json.Unmarshal(data, &f); ferr == nil {
			*n = jkFlexInt64(int64(f))
			return nil
		}
		return err
	}
	*n = jkFlexInt64(parsed)
	return nil
}

// Int64 returns the stored value.
func (n jkFlexInt64) Int64() int64 { return int64(n) }
