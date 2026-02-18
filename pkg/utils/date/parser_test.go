package date

import (
	"strings"
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

// ============================================================================
// Tests for Unix Timestamps
// ============================================================================

func TestParseDate_UnixTimestamp(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		checkYear int
	}{
		{
			name:      "seconds timestamp",
			input:     "1704067200", // 2024-01-01 00:00:00 UTC
			checkYear: 2024,
		},
		{
			name:      "milliseconds timestamp",
			input:     "1704067200000", // 2024-01-01 00:00:00 UTC
			checkYear: 2024,
		},
		{
			name:      "timestamp with noise",
			input:     "/1704067200/",
			checkYear: 2024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Year() != tt.checkYear {
				t.Errorf("expected year %d, got %d", tt.checkYear, result.Year())
			}
		})
	}
}

// ============================================================================
// Tests for Extended Standard Formats
// ============================================================================

func TestParseDate_ExtendedFormats(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "RFC1123Z",
			input: "Mon, 15 Jan 2024 10:30:00 -0700",
		},
		{
			name:  "compact",
			input: "20240115",
		},
		{
			name:  "US format",
			input: "01/15/2024",
		},
		{
			name:  "with weekday",
			input: "Monday, 15 Jan 2024",
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

// ============================================================================
// Tests for English Natural Language
// ============================================================================

func TestParseDate_EnglishNatural(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		checkFunc func(time.Time) bool
	}{
		{
			name:  "today",
			input: "today",
			checkFunc: func(tm time.Time) bool {
				return IsToday(tm)
			},
		},
		{
			name:  "tomorrow",
			input: "tomorrow",
			checkFunc: func(tm time.Time) bool {
				expected := time.Now().AddDate(0, 0, 1)
				return tm.Day() == expected.Day()
			},
		},
		{
			name:  "Monday",
			input: "Monday",
			checkFunc: func(tm time.Time) bool {
				return tm.Weekday() == time.Monday
			},
		},
		{
			name:  "Mon abbreviated",
			input: "Mon",
			checkFunc: func(tm time.Time) bool {
				return tm.Weekday() == time.Monday
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tt.checkFunc(result) {
				t.Errorf("check failed for input %q, got %v", tt.input, result)
			}
		})
	}
}

// ============================================================================
// Tests for Chinese Dates
// ============================================================================

func TestParseDate_ChineseBasic(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		input     string
		checkFunc func(time.Time) bool
	}{
		{
			name:  "今天",
			input: "今天",
			checkFunc: func(tm time.Time) bool {
				return IsToday(tm)
			},
		},
		{
			name:  "明天",
			input: "明天",
			checkFunc: func(tm time.Time) bool {
				expected := now.AddDate(0, 0, 1)
				return tm.Day() == expected.Day()
			},
		},
		{
			name:  "后天",
			input: "后天",
			checkFunc: func(tm time.Time) bool {
				expected := now.AddDate(0, 0, 2)
				return tm.Day() == expected.Day()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tt.checkFunc(result) {
				t.Errorf("check failed for input %q, got %v", tt.input, result)
			}
		})
	}
}

func TestParseDate_ChineseWeekdays(t *testing.T) {
	inputs := []string{
		"周一", "周二", "周三", "周四", "周五", "周六", "周日",
		"星期一", "星期二", "星期三", "星期四", "星期五", "星期六", "星期日",
		"礼拜一", "礼拜二", "礼拜三", "礼拜四", "礼拜五", "礼拜六", "礼拜日",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			result, err := ParseDate(input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.IsZero() {
				t.Error("expected non-zero time, got zero")
			}
		})
	}
}

func TestParseDate_ChineseRelative(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		checkFunc func(time.Time) bool
	}{
		{
			name:  "3秒前",
			input: "3秒前",
			checkFunc: func(tm time.Time) bool {
				return time.Since(tm) < 10*time.Second
			},
		},
		{
			name:  "5分钟前",
			input: "5分钟前",
			checkFunc: func(tm time.Time) bool {
				return time.Since(tm) < 10*time.Minute
			},
		},
		{
			name:  "2小时前",
			input: "2小时前",
			checkFunc: func(tm time.Time) bool {
				return time.Since(tm) < 3*time.Hour
			},
		},
		{
			name:  "3天前",
			input: "3天前",
			checkFunc: func(tm time.Time) bool {
				expected := time.Now().AddDate(0, 0, -3)
				return tm.Day() == expected.Day()
			},
		},
		{
			name:  "1周前",
			input: "1周前",
			checkFunc: func(tm time.Time) bool {
				expected := time.Now().AddDate(0, 0, -7)
				return tm.Day() == expected.Day()
			},
		},
		{
			name:  "2月前",
			input: "2月前",
			checkFunc: func(tm time.Time) bool {
				expected := time.Now().AddDate(0, -2, 0)
				return tm.Month() == expected.Month()
			},
		},
		{
			name:  "1年前",
			input: "1年前",
			checkFunc: func(tm time.Time) bool {
				expected := time.Now().AddDate(-1, 0, 0)
				return tm.Year() == expected.Year()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tt.checkFunc(result) {
				t.Errorf("check failed for input %q", tt.input)
			}
		})
	}
}

func TestParseDate_ChineseDateFormat(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		checkYear  int
		checkMonth int
		checkDay   int
	}{
		{
			name:       "2024年01月02日",
			input:      "2024年01月02日",
			checkYear:  2024,
			checkMonth: 1,
			checkDay:   2,
		},
		{
			name:       "2024年12月25日",
			input:      "2024年12月25日",
			checkYear:  2024,
			checkMonth: 12,
			checkDay:   25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Year() != tt.checkYear {
				t.Errorf("expected year %d, got %d", tt.checkYear, result.Year())
			}
			if int(result.Month()) != tt.checkMonth {
				t.Errorf("expected month %d, got %d", tt.checkMonth, result.Month())
			}
			if result.Day() != tt.checkDay {
				t.Errorf("expected day %d, got %d", tt.checkDay, result.Day())
			}
		})
	}
}

func TestParseDate_ChinesePeriods(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		input     string
		checkFunc func(time.Time) bool
	}{
		{
			name:  "本周",
			input: "本周",
			checkFunc: func(tm time.Time) bool {
				// Should be start of week (Monday)
				return tm.Weekday() == time.Monday
			},
		},
		{
			name:  "这周",
			input: "这周",
			checkFunc: func(tm time.Time) bool {
				return tm.Weekday() == time.Monday
			},
		},
		{
			name:  "本月",
			input: "本月",
			checkFunc: func(tm time.Time) bool {
				return tm.Day() == 1
			},
		},
		{
			name:  "今年",
			input: "今年",
			checkFunc: func(tm time.Time) bool {
				return tm.Year() == now.Year() && tm.Day() == 1 && tm.Month() == time.January
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDate(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tt.checkFunc(result) {
				t.Errorf("check failed for input %q", tt.input)
			}
		})
	}
}

// ============================================================================
// Tests for Utility Functions
// ============================================================================

func TestIsValidDate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid ISO8601",
			input:    "2024-01-15T10:00:00Z",
			expected: true,
		},
		{
			name:     "valid relative",
			input:    "yesterday",
			expected: true,
		},
		{
			name:     "valid Chinese",
			input:    "昨天",
			expected: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: true,
		},
		{
			name:     "invalid",
			input:    "not a date",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidDate(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFormatDate(t *testing.T) {
	tm := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	result := FormatDate(tm)

	if !strings.Contains(result, "2024-01-15") {
		t.Errorf("expected date in result, got '%s'", result)
	}
}

func TestFormatDateCustom(t *testing.T) {
	tm := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	result := FormatDateCustom(tm, "2006/01/02")

	expected := "2024/01/15"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestAgeInDays(t *testing.T) {
	past := time.Now().Add(-72 * time.Hour) // 3 days ago
	days := AgeInDays(past)

	if days != 3 {
		t.Errorf("expected 3 days, got %d", days)
	}
}

func TestAgeInHours(t *testing.T) {
	past := time.Now().Add(-5 * time.Hour)
	hours := AgeInHours(past)

	if hours != 5 {
		t.Errorf("expected 5 hours, got %d", hours)
	}
}

func TestIsToday(t *testing.T) {
	now := time.Now()
	if !IsToday(now) {
		t.Error("expected now to be today")
	}

	yesterday := time.Now().AddDate(0, 0, -1)
	if IsToday(yesterday) {
		t.Error("expected yesterday to not be today")
	}
}

func TestIsYesterday(t *testing.T) {
	yesterday := time.Now().AddDate(0, 0, -1)
	if !IsYesterday(yesterday) {
		t.Error("expected yesterday to be detected")
	}

	today := time.Now()
	if IsYesterday(today) {
		t.Error("expected today to not be yesterday")
	}
}

func TestStartOfDay(t *testing.T) {
	now := time.Now()
	start := StartOfDay(now)

	if start.Hour() != 0 || start.Minute() != 0 || start.Second() != 0 {
		t.Errorf("expected 00:00:00, got %02d:%02d:%02d", start.Hour(), start.Minute(), start.Second())
	}

	if start.Day() != now.Day() {
		t.Error("start of day should have same day as input")
	}
}

func TestEndOfDay(t *testing.T) {
	now := time.Now()
	end := EndOfDay(now)

	if end.Hour() != 23 || end.Minute() != 59 || end.Second() != 59 {
		t.Errorf("expected 23:59:59, got %02d:%02d:%02d", end.Hour(), end.Minute(), end.Second())
	}
}

func TestStartOfWeek(t *testing.T) {
	now := time.Now()
	start := StartOfWeek(now)

	if start.Weekday() != time.Monday {
		t.Errorf("expected Monday, got %v", start.Weekday())
	}
}

func TestStartOfMonth(t *testing.T) {
	now := time.Now()
	start := StartOfMonth(now)

	if start.Day() != 1 {
		t.Errorf("expected day 1, got %d", start.Day())
	}

	if start.Month() != now.Month() {
		t.Error("start of month should have same month as input")
	}
}

func TestStartOfYear(t *testing.T) {
	now := time.Now()
	start := StartOfYear(now)

	if start.Day() != 1 || start.Month() != 1 {
		t.Error("start of year should be Jan 1")
	}

	if start.Year() != now.Year() {
		t.Error("start of year should have same year as input")
	}
}

