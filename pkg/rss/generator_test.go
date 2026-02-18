package rss

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/xihale/rsshub-go/pkg/models"
)

func TestGenerateRSS(t *testing.T) {
	feed := &models.Feed{
		Title:       "Test Feed",
		Link:        "https://example.com",
		Description: "A test feed",
		Items: []models.Item{
			{
				Title:       "Test Item",
				Link:        "https://example.com/item1",
				Description: "<p>Test content</p>",
				GUID:        "item1",
				PubDate:     time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			},
		},
	}

	data, err := GenerateRSS(feed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that it's valid XML
	var rss RSSFeed
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("failed to parse RSS XML: %v", err)
	}

	if rss.Version != "2.0" {
		t.Errorf("expected version 2.0, got %s", rss.Version)
	}

	if rss.Channel.Title != "Test Feed" {
		t.Errorf("expected title 'Test Feed', got %s", rss.Channel.Title)
	}

	if len(rss.Channel.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(rss.Channel.Items))
	}

	item := rss.Channel.Items[0]
	if item.Title != "Test Item" {
		t.Errorf("expected item title 'Test Item', got %s", item.Title)
	}

	if item.GUID != "item1" {
		t.Errorf("expected GUID 'item1', got %s", item.GUID)
	}
}

func TestGenerateRSS_WithAuthor(t *testing.T) {
	feed := &models.Feed{
		Title:       "Test Feed",
		Link:        "https://example.com",
		Description: "A test feed",
		Author: &models.Author{
			Name:  "John Doe",
			Email: "john@example.com",
		},
		Items: []models.Item{
			{
				Title:   "Test Item",
				Link:    "https://example.com/item1",
				GUID:    "item1",
				PubDate: time.Now(),
			},
		},
	}

	data, err := GenerateRSS(feed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var rss RSSFeed
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("failed to parse RSS XML: %v", err)
	}

	if rss.Channel.ManagingEditor != "john@example.com" {
		t.Errorf("expected managing editor 'john@example.com', got %s", rss.Channel.ManagingEditor)
	}
}

func TestGenerateRSS_WithCategories(t *testing.T) {
	feed := &models.Feed{
		Title:       "Test Feed",
		Link:        "https://example.com",
		Description: "A test feed",
		Items: []models.Item{
			{
				Title:      "Test Item",
				Link:       "https://example.com/item1",
				GUID:       "item1",
				Categories: []string{"tech", "news"},
				PubDate:    time.Now(),
			},
		},
	}

	data, err := GenerateRSS(feed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var rss RSSFeed
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("failed to parse RSS XML: %v", err)
	}

	item := rss.Channel.Items[0]
	if len(item.Categories) != 2 {
		t.Errorf("expected 2 categories, got %d", len(item.Categories))
	}
}

func TestGenerateRSS_XMLEncoding(t *testing.T) {
	feed := &models.Feed{
		Title:       "Test Feed",
		Link:        "https://example.com",
		Description: "A test feed",
		Items: []models.Item{
			{
				Title:       "Item with <html> tags",
				Link:        "https://example.com/item1",
				Description: `<script>alert("test")</script>`,
				GUID:        "item1",
				PubDate:     time.Now(),
			},
		},
	}

	data, err := GenerateRSS(feed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain XML declaration
	if !strings.HasPrefix(string(data), `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("expected XML declaration")
	}

	// Should contain generator comment
	if !strings.Contains(string(data), `generator="RSSHub-Go/1.0"`) {
		t.Error("expected generator comment")
	}
}

func TestGenerateAtom(t *testing.T) {
	feed := &models.Feed{
		Title:       "Test Feed",
		Link:        "https://example.com",
		Description: "A test feed",
		Items: []models.Item{
			{
				Title:       "Test Item",
				Link:        "https://example.com/item1",
				Description: "<p>Test content</p>",
				GUID:        "item1",
				PubDate:     time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			},
		},
	}

	data, err := GenerateAtom(feed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that it's valid XML
	var atom AtomFeed
	if err := xml.Unmarshal(data, &atom); err != nil {
		t.Fatalf("failed to parse Atom XML: %v", err)
	}

	if atom.Title != "Test Feed" {
		t.Errorf("expected title 'Test Feed', got %s", atom.Title)
	}

	if len(atom.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(atom.Entries))
	}

	entry := atom.Entries[0]
	if entry.Title != "Test Item" {
		t.Errorf("expected entry title 'Test Item', got %s", entry.Title)
	}

	if entry.ID != "item1" {
		t.Errorf("expected ID 'item1', got %s", entry.ID)
	}
}

func TestGenerateAtom_WithAuthor(t *testing.T) {
	feed := &models.Feed{
		Title:       "Test Feed",
		Link:        "https://example.com",
		Description: "A test feed",
		Author: &models.Author{
			Name:  "John Doe",
			Email: "john@example.com",
		},
		Items: []models.Item{
			{
				Title:   "Test Item",
				Link:    "https://example.com/item1",
				GUID:    "item1",
				PubDate: time.Now(),
			},
		},
	}

	data, err := GenerateAtom(feed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var atom AtomFeed
	if err := xml.Unmarshal(data, &atom); err != nil {
		t.Fatalf("failed to parse Atom XML: %v", err)
	}

	if len(atom.Authors) != 1 {
		t.Fatalf("expected 1 author, got %d", len(atom.Authors))
	}

	if atom.Authors[0].Name != "John Doe" {
		t.Errorf("expected author name 'John Doe', got %s", atom.Authors[0].Name)
	}
}

func TestGenerateAtom_Links(t *testing.T) {
	feed := &models.Feed{
		Title:       "Test Feed",
		Link:        "https://example.com/feed",
		Description: "A test feed",
		Items: []models.Item{
			{
				Title:   "Test Item",
				Link:    "https://example.com/item1",
				GUID:    "item1",
				PubDate: time.Now(),
			},
		},
	}

	data, err := GenerateAtom(feed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var atom AtomFeed
	if err := xml.Unmarshal(data, &atom); err != nil {
		t.Fatalf("failed to parse Atom XML: %v", err)
	}

	// Should have self and alternate links
	if len(atom.Links) != 2 {
		t.Errorf("expected 2 links, got %d", len(atom.Links))
	}
}

func TestGenerateAtom_WithCategories(t *testing.T) {
	feed := &models.Feed{
		Title:       "Test Feed",
		Link:        "https://example.com",
		Description: "A test feed",
		Items: []models.Item{
			{
				Title:      "Test Item",
				Link:       "https://example.com/item1",
				GUID:       "item1",
				Categories: []string{"tech", "news"},
				PubDate:    time.Now(),
			},
		},
	}

	data, err := GenerateAtom(feed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var atom AtomFeed
	if err := xml.Unmarshal(data, &atom); err != nil {
		t.Fatalf("failed to parse Atom XML: %v", err)
	}

	entry := atom.Entries[0]
	if len(entry.Categories) != 2 {
		t.Errorf("expected 2 categories, got %d", len(entry.Categories))
	}
}

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic HTML",
			input:    `<p>Hello & goodbye</p>`,
			expected: `&lt;p&gt;Hello &amp; goodbye&lt;/p&gt;`,
		},
		{
			name:     "special chars",
			input:    `<>&"'`,
			expected: `&lt;&gt;&amp;&#34;&#39;`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeHTML(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGenerateRSS_EmptyFeed(t *testing.T) {
	feed := &models.Feed{
		Title:       "Empty Feed",
		Link:        "https://example.com",
		Description: "A feed with no items",
		Items:       []models.Item{},
	}

	data, err := GenerateRSS(feed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var rss RSSFeed
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("failed to parse RSS XML: %v", err)
	}

	if len(rss.Channel.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(rss.Channel.Items))
	}
}

func TestGenerateAtom_EmptyFeed(t *testing.T) {
	feed := &models.Feed{
		Title:       "Empty Feed",
		Link:        "https://example.com",
		Description: "A feed with no items",
		Items:       []models.Item{},
	}

	data, err := GenerateAtom(feed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var atom AtomFeed
	if err := xml.Unmarshal(data, &atom); err != nil {
		t.Fatalf("failed to parse Atom XML: %v", err)
	}

	if len(atom.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(atom.Entries))
	}
}
