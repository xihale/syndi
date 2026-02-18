package date

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParseDate parses various date formats including natural language
// Supports English, Chinese, ISO 8601, Unix timestamps, and more
func ParseDate(input string) (time.Time, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return time.Time{}, nil
	}

	// 1. Try Unix timestamp (seconds or milliseconds)
	if t, ok := parseTimestamp(input); ok {
		return t, nil
	}

	// 2. Try standard ISO 8601 and common formats
	if t, err := parseStandard(input); err == nil {
		return t, nil
	}

	// 3. Try natural language patterns
	if t, err := parseNaturalLanguage(input); err == nil {
		return t, nil
	}

	// 4. Try Chinese-specific patterns
	if t, err := parseChinese(input); err == nil {
		return t, nil
	}

	return time.Time{}, nil
}

// parseTimestamp attempts to parse Unix timestamp
func parseTimestamp(input string) (time.Time, bool) {
	// Remove any non-digit characters
	clean := strings.TrimFunc(input, func(r rune) bool {
		return r < '0' || r > '9'
	})

	if clean == "" {
		return time.Time{}, false
	}

	// Try as seconds
	if seconds, err := strconv.ParseInt(clean, 10, 64); err == nil {
		// Check if it's seconds (10 digits) or milliseconds (13 digits)
		if seconds > 1000000000000 { // Milliseconds
			return time.Unix(seconds/1000, (seconds%1000)*1000000), true
		}
		// Seconds
		return time.Unix(seconds, 0), true
	}

	return time.Time{}, false
}

// parseStandard tries common standard date formats
func parseStandard(input string) (time.Time, error) {
	// Extended list of standard formats
	formats := []string{
		// ISO 8601 variants
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05.999Z",
		"2006-01-02T15:04:05.999-07:00",

		// RFC formats
		time.RFC1123,
		time.RFC1123Z,
		time.RFC822,
		time.RFC822Z,

		// Common date formats
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006/01/02 15:04:05",
		"2006/01/02 15:04",
		"2006-01-02",
		"2006/01/02",

		// Month name variants
		"02 Jan 2006 15:04:05",
		"02 Jan 2006",
		"Jan 2, 2006 15:04:05",
		"Jan 2, 2006",
		"January 2, 2006",
		"02-Jan-2006",
		"2006-Jan-02",

		// Compact formats
		"20060102",
		"20060102T150405Z",

		// US format
		"01/02/2006",
		"01-02-2006",

		// With weekday
		"Mon, 02 Jan 2006 15:04:05",
		"Monday, 02 Jan 2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, input); err == nil {
			return t, nil
		}
	}

	return time.Time{}, &time.ParseError{}
}

// parseNaturalLanguage parses English natural language dates
func parseNaturalLanguage(input string) (time.Time, error) {
	lower := strings.ToLower(input)

	// "now" or "just now"
	if lower == "now" || lower == "just now" || lower == "today" {
		return time.Now(), nil
	}

	// "tomorrow" and "yesterday"
	if lower == "yesterday" {
		return time.Now().AddDate(0, 0, -1), nil
	}
	if lower == "tomorrow" {
		return time.Now().AddDate(0, 0, 1), nil
	}

	// Weekday names (this week)
	weekdayMap := map[string]time.Weekday{
		"sunday":    time.Sunday,
		"monday":    time.Monday,
		"tuesday":   time.Tuesday,
		"wednesday": time.Wednesday,
		"thursday":  time.Thursday,
		"friday":    time.Friday,
		"saturday":  time.Saturday,
		"sun":       time.Sunday,
		"mon":       time.Monday,
		"tue":       time.Tuesday,
		"wed":       time.Wednesday,
		"thu":       time.Thursday,
		"fri":       time.Friday,
		"sat":       time.Saturday,
	}

	for weekdayName, targetDay := range weekdayMap {
		if strings.Contains(lower, weekdayName) {
			now := time.Now()
			currentDay := now.Weekday()
			daysUntil := int(targetDay - currentDay + 7) % 7

			// If it's "last <weekday>" or "last <day>", go back
			if strings.HasPrefix(lower, "last ") {
				daysUntil -= 7
			}

			return now.AddDate(0, 0, daysUntil), nil
		}
	}

	// Relative time: "3 hours ago", "2 days ago"
	re := regexp.MustCompile(`(\d+)\s*(second|minute|hour|day|week|month|year)s?\s+ago`)
	if matches := re.FindStringSubmatch(lower); len(matches) == 3 {
		num, _ := strconv.Atoi(matches[1])
		unit := matches[2]

		now := time.Now()
		switch unit {
		case "second":
			return now.Add(time.Duration(-num) * time.Second), nil
		case "minute":
			return now.Add(time.Duration(-num) * time.Minute), nil
		case "hour":
			return now.Add(time.Duration(-num) * time.Hour), nil
		case "day":
			return now.AddDate(0, 0, -num), nil
		case "week":
			return now.AddDate(0, 0, -num*7), nil
		case "month":
			return now.AddDate(0, -num, 0), nil
		case "year":
			return now.AddDate(-num, 0, 0), nil
		}
	}

	// "last week", "last month", "last year"
	if lower == "last week" {
		return time.Now().AddDate(0, 0, -7), nil
	}
	if lower == "last month" {
		return time.Now().AddDate(0, -1, 0), nil
	}
	if lower == "last year" {
		return time.Now().AddDate(-1, 0, 0), nil
	}

	return time.Time{}, &time.ParseError{}
}

// parseChinese parses Chinese date formats
func parseChinese(input string) (time.Time, error) {
	now := time.Now()

	// Today, yesterday, day before yesterday
	if input == "今天" || strings.Contains(input, "今天") {
		return now, nil
	}
	if strings.Contains(input, "昨天") {
		return now.AddDate(0, 0, -1), nil
	}
	if strings.Contains(input, "前天") {
		return now.AddDate(0, 0, -2), nil
	}

	// Tomorrow
	if strings.Contains(input, "明天") {
		return now.AddDate(0, 0, 1), nil
	}
	if strings.Contains(input, "后天") {
		return now.AddDate(0, 0, 2), nil
	}

	// Weekday names in Chinese
	weekdayMap := map[string]time.Weekday{
		"周日": time.Sunday,
		"周一": time.Monday,
		"周二": time.Tuesday,
		"周三": time.Wednesday,
		"周四": time.Thursday,
		"周五": time.Friday,
		"周六": time.Saturday,
		"星期日": time.Sunday,
		"星期一": time.Monday,
		"星期二": time.Tuesday,
		"星期三": time.Wednesday,
		"星期四": time.Thursday,
		"星期五": time.Friday,
		"星期六": time.Saturday,
		"礼拜日": time.Sunday,
		"礼拜一": time.Monday,
		"礼拜二": time.Tuesday,
		"礼拜三": time.Wednesday,
		"礼拜四": time.Thursday,
		"礼拜五": time.Friday,
		"礼拜六": time.Saturday,
	}

	for weekdayName, targetDay := range weekdayMap {
		if strings.Contains(input, weekdayName) {
			currentDay := now.Weekday()
			daysUntil := int(targetDay-currentDay+7) % 7
			return now.AddDate(0, 0, daysUntil), nil
		}
	}

	// Chinese relative time: "3小时前", "2天前"
	re := regexp.MustCompile(`(\d+)\s*(秒|分钟|小时|天|周|月|年)前`)
	if matches := re.FindStringSubmatch(input); len(matches) == 3 {
		num, _ := strconv.Atoi(matches[1])
		unit := matches[2]

		switch unit {
		case "秒":
			return now.Add(time.Duration(-num) * time.Second), nil
		case "分钟":
			return now.Add(time.Duration(-num) * time.Minute), nil
		case "小时":
			return now.Add(time.Duration(-num) * time.Hour), nil
		case "天":
			return now.AddDate(0, 0, -num), nil
		case "周":
			return now.AddDate(0, 0, -num*7), nil
		case "月":
			return now.AddDate(0, -num, 0), nil
		case "年":
			return now.AddDate(-num, 0, 0), nil
		}
	}

	// Chinese date format: 2024年01月02日
	re = regexp.MustCompile(`(\d{4})年(\d{1,2})月(\d{1,2})日`)
	if matches := re.FindStringSubmatch(input); len(matches) == 4 {
		year, _ := strconv.Atoi(matches[1])
		month, _ := strconv.Atoi(matches[2])
		day, _ := strconv.Atoi(matches[3])
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local), nil
	}

	// Chinese date format with time: 2024年01月02日 15时04分
	re = regexp.MustCompile(`(\d{4})年(\d{1,2})月(\d{1,2})日\s*(\d{1,2})时(\d{1,2})分`)
	if matches := re.FindStringSubmatch(input); len(matches) == 6 {
		year, _ := strconv.Atoi(matches[1])
		month, _ := strconv.Atoi(matches[2])
		day, _ := strconv.Atoi(matches[3])
		hour, _ := strconv.Atoi(matches[4])
		minute, _ := strconv.Atoi(matches[5])
		return time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.Local), nil
	}

	// "本周", "上周", "下周"
	if strings.Contains(input, "本周") || strings.Contains(input, "这周") {
		// Start of this week (Monday)
		daysSinceMonday := int(now.Weekday() - time.Monday + 7) % 7
		return now.AddDate(0, 0, -daysSinceMonday), nil
	}
	if strings.Contains(input, "上周") {
		// Start of last week
		daysSinceMonday := int(now.Weekday() - time.Monday + 7) % 7
		return now.AddDate(0, 0, -daysSinceMonday-7), nil
	}
	if strings.Contains(input, "下周") {
		// Start of next week
		daysSinceMonday := int(now.Weekday() - time.Monday + 7) % 7
		return now.AddDate(0, 0, -daysSinceMonday+7), nil
	}

	// "本月", "上月", "下月"
	if strings.Contains(input, "本月") || strings.Contains(input, "这个月") {
		// Start of this month
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local), nil
	}
	if strings.Contains(input, "上月") || strings.Contains(input, "上个月") {
		// Start of last month
		return time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.Local), nil
	}
	if strings.Contains(input, "下月") || strings.Contains(input, "下个月") {
		// Start of next month
		return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.Local), nil
	}

	// "今年", "去年", "明年"
	if strings.Contains(input, "今年") {
		// Start of this year
		return time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.Local), nil
	}
	if strings.Contains(input, "去年") || strings.Contains(input, "上年") {
		// Start of last year
		return time.Date(now.Year()-1, 1, 1, 0, 0, 0, 0, time.Local), nil
	}
	if strings.Contains(input, "明年") {
		// Start of next year
		return time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, time.Local), nil
	}

	return time.Time{}, &time.ParseError{}
}

// ParseDateInLocation parses a date in a specific location
func ParseDateInLocation(input string, loc *time.Location) (time.Time, error) {
	t, err := ParseDate(input)
	if err != nil {
		return time.Time{}, err
	}
	return t.In(loc), nil
}

// IsValidDate checks if a date string can be parsed
func IsValidDate(input string) bool {
	_, err := ParseDate(input)
	return err == nil
}

// FormatDate formats a time to a standard string
// Default format: RFC3339
func FormatDate(t time.Time) string {
	return t.Format(time.RFC3339)
}

// FormatDateCustom formats a time using a custom format string
func FormatDateCustom(t time.Time, format string) string {
	return t.Format(format)
}

// Age calculates the age (duration) since the given time
func Age(t time.Time) time.Duration {
	return time.Since(t)
}

// AgeInDays returns the number of days since the given time
func AgeInDays(t time.Time) int {
	return int(time.Since(t).Hours() / 24)
}

// AgeInHours returns the number of hours since the given time
func AgeInHours(t time.Time) int {
	return int(time.Since(t).Hours())
}

// IsToday checks if the given time is today
func IsToday(t time.Time) bool {
	now := time.Now()
	return t.Year() == now.Year() && t.Month() == now.Month() && t.Day() == now.Day()
}

// IsYesterday checks if the given time is yesterday
func IsYesterday(t time.Time) bool {
	yesterday := time.Now().AddDate(0, 0, -1)
	return t.Year() == yesterday.Year() && t.Month() == yesterday.Month() && t.Day() == yesterday.Day()
}

// StartOfDay returns the start of the day (00:00:00)
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay returns the end of the day (23:59:59)
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

// StartOfWeek returns the start of the week (Monday)
func StartOfWeek(t time.Time) time.Time {
	daysSinceMonday := int(t.Weekday() - time.Monday + 7) % 7
	return t.AddDate(0, 0, -daysSinceMonday)
}

// StartOfMonth returns the start of the month
func StartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// StartOfYear returns the start of the year
func StartOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
}
