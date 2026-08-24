package routes

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"
)

const (
	wikipediaAPIBase = "https://api.wikimedia.org/feed/v1/wikipedia/en"
	wikipediaBaseURL = "https://en.wikipedia.org/"
)

// wikiPage is a Wikipedia page summary as returned by the Wikimedia feed API.
type wikiPage struct {
	Title  string `json:"title"` // canonical underscored title
	Titles struct {
		Canonical  string `json:"canonical"`
		Normalized string `json:"normalized"`
	} `json:"titles"`
	ContentURLs struct {
		Desktop struct {
			Page string `json:"page"`
		} `json:"desktop"`
	} `json:"content_urls"`
	Thumbnail struct {
		Source string `json:"source"`
	} `json:"thumbnail"`
	Extract string `json:"extract"`
	Views   int    `json:"views"` // present on most-read articles
	Rank    int    `json:"rank"`  // present on most-read articles
}

// link returns the desktop article URL, falling back to one constructed from
// the canonical title.
func (p wikiPage) link() string {
	if p.ContentURLs.Desktop.Page != "" {
		return p.ContentURLs.Desktop.Page
	}
	title := p.Titles.Canonical
	if title == "" {
		title = p.Title
	}
	if title == "" {
		return ""
	}
	return "https://en.wikipedia.org/wiki/" + title
}

// truncateTitle shortens long text into a readable item title.
func truncateTitle(text string, max int) string {
	s := strings.TrimSpace(text)
	if len(s) <= max {
		return s
	}
	cut := s[:max]
	if idx := strings.LastIndexByte(cut, ' '); idx > max/2 {
		cut = cut[:idx]
	}
	return cut + "…"
}

// atoi parses a two-digit string, returning 0 on failure.
func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// hashString returns a short FNV-1a hash for GUID stability.
func hashString(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}

// validateMonthDay validates an MM-DD parameter and returns its parts.
func validateMonthDay(monthday string) (mm, dd string, err error) {
	parts := strings.Split(monthday, "-")
	if len(parts) != 2 || len(parts[0]) != 2 || len(parts[1]) != 2 {
		return "", "", fmt.Errorf("invalid monthday %q, expected MM-DD format like 08-24", monthday)
	}
	var m, d int
	if _, err := fmt.Sscanf(parts[0], "%02d", &m); err != nil || m < 1 || m > 12 {
		return "", "", fmt.Errorf("invalid month in %q", monthday)
	}
	if _, err := fmt.Sscanf(parts[1], "%02d", &d); err != nil || d < 1 || d > 31 {
		return "", "", fmt.Errorf("invalid day in %q", monthday)
	}
	return parts[0], parts[1], nil
}

// onThisDayEventDate builds a date from the historical year and requested month/day.
func onThisDayEventDate(year, month, day int) time.Time {
	if year == 0 {
		return time.Time{}
	}
	return time.Date(year, time.Month(month), day, 12, 0, 0, 0, time.UTC)
}
