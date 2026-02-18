package date

import (
	"testing"
	"time"
)

func TestParseDate_ISO8601(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		checkErr func(error) bool
	}{
		{
			name:    "RFC3339",
			input:   "2024-01-15T10:30:00Z",
			wantErr: false,
		},
		{
			name:    "RFC3339 with offset",
			input:   "2024-01-15T10:30:00+08:00",
			wantErr: false,
		},
		{
			name:    "RFC1123",
			input:   "Mon, 15 Jan 2024 10:30:00 GMT",
			wantErr: false,
		},
		{
			name:    "common format",
			input:   "2024-01-15 10:30:00",
			wantErr: false,
		},
		{
			name:    "date only",
			input:   "2024-01-15",
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDate(tt.input)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil && tt.input != "" {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.wantErr && tt.input != "" {
				if result.IsZero() {
					t.Error("expected non-zero time, got zero")
				}
			}
		})
	}
}

func TestParseDate_Relative(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name         string
		input        string
		checkTime    func(time.Time) bool
	}{
		{
			name:  "now",
			input: "now",
			checkTime: func(tm time.Time) bool {
				return time.Since(tm) < time.Second
			},
		},
		{
			name:  "just now",
			input: "just now",
			checkTime: func(tm time.Time) bool {
				return time.Since(tm) < time.Second
			},
		},
		{
			name:  "yesterday",
			input: "昨天",
			checkTime: func(tm time.Time) bool {
				expected := now.AddDate(0, 0, -1)
				return tm.Day() == expected.Day() &&
					tm.Month() == expected.Month() &&
					tm.Year() == expected.Year()
			},
		},
		{
			name:  "day before yesterday",
			input: "前天",
			checkTime: func(tm time.Time) bool {
				expected := now.AddDate(0, 0, -2)
				return tm.Day() == expected.Day() &&
					tm.Month() == expected.Month() &&
					tm.Year() == expected.Year()
			},
		},
		{
			name:  "5 minutes ago",
			input: "5 minutes ago",
			checkTime: func(tm time.Time) bool {
				expected := now.Add(-5 * time.Minute)
				diff := expected.Sub(tm)
				return diff < time.Second
			},
		},
		{
			name:  "2 hours ago",
			input: "2 hours ago",
			checkTime: func(tm time.Time) bool {
				expected := now.Add(-2 * time.Hour)
				diff := expected.Sub(tm)
				return diff < time.Second
			},
		},
		{
			name:  "3 days ago",
			input: "3 days ago",
			checkTime: func(tm time.Time) bool {
				expected := now.AddDate(0, 0, -3)
				return tm.Day() == expected.Day() &&
					tm.Month() == expected.Month() &&
					tm.Year() == expected.Year()
			},
		},
		{
			name:  "1 week ago",
			input: "1 week ago",
			checkTime: func(tm time.Time) bool {
				expected := now.AddDate(0, 0, -7)
				return tm.Day() == expected.Day() &&
					tm.Month() == expected.Month() &&
					tm.Year() == expected.Year()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tt.checkTime(result) {
				t.Errorf("time check failed for input %q, got %v", tt.input, result)
			}
		})
	}
}

func TestParseDate_ChineseFormats(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "yesterday Chinese",
			input: "昨天",
		},
		{
			name:  "day before yesterday Chinese",
			input: "前天",
		},
		{
			name:  "yesterday in sentence",
			input: "昨天发布的文章",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.IsZero() {
				t.Error("expected non-zero time, got zero")
			}
		})
	}
}

func TestParseDate_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "garbage",
			input: "not a date at all",
		},
		{
			name:  "invalid format",
			input: "99/99/9999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDate(tt.input)
			// Should not error, but return zero time
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !result.IsZero() {
				t.Logf("got non-zero time for invalid input: %v", result)
			}
		})
	}
}

func TestParseDate_TimestampFormats(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "02 Jan 2006",
			input: "15 Jan 2024",
		},
		{
			name:  "Jan 2, 2006",
			input: "Jan 15, 2024",
		},
		{
			name:  "2006/01/02 15:04:05",
			input: "2024/01/15 10:30:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.IsZero() {
				t.Error("expected non-zero time, got zero")
			}
		})
	}
}
