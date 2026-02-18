package models

import (
	"time"

	ctxpkg "github.com/rsshub/go/pkg/context"
)

// Feed represents an RSS/Atom feed
type Feed struct {
	Title       string
	Description string
	Link        string
	Items       []Item
	Author      *Author `json:"author,omitempty"`
	Updated     *time.Time `json:"updated,omitempty"`
}

// Item represents a single entry in a feed
type Item struct {
	Title       string
	Description string
	Link        string
	PubDate     time.Time
	GUID        string
	Author      *Author `json:"author,omitempty"`
	Updated     *time.Time `json:"updated,omitempty"`
	Categories  []string `json:"categories,omitempty"`
}

// Author represents an author
type Author struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	URI   string `json:"uri,omitempty"`
}

// Route represents a registered route handler
type Route struct {
	Path         string
	Name         string
	URL          string
	Handler      HandlerFunc
	Example      string
	Maintainers  []string
	Description  string
	Categories   []Category
	Features     Features
	Parameters   []Parameter
	Radar        []RadarItem
	CacheTTL     *time.Duration `json:"cacheTTL,omitempty"` // Optional cache TTL override
}

// HandlerFunc is the function signature for route handlers
type HandlerFunc func(*ctxpkg.Context) (*Feed, error)

// Category represents a route category
type Category struct {
	Name        string
	Description string
}

// Features defines optional route capabilities and requirements
type Features struct {
	RequireConfig   bool   `json:"requireConfig,omitempty"`
	RequirePassword bool   `json:"requirePassword,omitempty"`
	AntiCrawler     bool   `json:"antiCrawler,omitempty"`
	SupportRadar    bool   `json:"supportRadar,omitempty"`
	NSFW            bool   `json:"nsfw,omitempty"`
}

// Parameter defines a query parameter for a route
type Parameter struct {
	Name         string
	Required     bool
	Default      string
	Description  string
	Values       []string
	Multivalued  bool
}

// RadarItem defines webSub hub/subscription rules
type RadarItem struct {
	Title      string   `json:"title,omitempty"`
	Source     []string `json:"source"`
	Target     string   `json:"target"`
	Weight     int      `json:"weight,omitempty"`
	Method     string   `json:"method,omitempty"`
	Body       string   `json:"body,omitempty"`
	Link       string   `json:"link,omitempty"`
	Description string  `json:"description,omitempty"`
}
