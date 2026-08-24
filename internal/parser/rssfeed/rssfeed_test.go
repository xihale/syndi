package rssfeed

import (
	"testing"
	"time"

	"github.com/xihale/rsshub-go/pkg/models"
)

func TestParseRSS2(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:content="http://purl.org/rss/1.0/modules/content/">
<channel>
  <title>Example News</title>
  <link>https://example.org/</link>
  <description>Example feed</description>
  <item>
    <title><![CDATA[First post]]></title>
    <link>https://example.org/posts/1</link>
    <guid isPermaLink="false">post-1</guid>
    <description>Hello &lt;b&gt;world&lt;/b></description>
    <content:encoded><![CDATA[<p>Full body text</p>]]></content:encoded>
    <pubDate>Tue, 10 Mar 2026 08:00:00 GMT</pubDate>
    <category>alpha</category>
    <category>beta</category>
  </item>
  <item>
    <title>Second post</title>
    <link>https://example.org/posts/2</link>
    <dc:creator>Jane Doe</dc:creator>
    <pubDate>Wed, 11 Mar 2026 09:30:00 +0800</pubDate>
  </item>
</channel>
</rss>`)

	feed, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if feed.Title != "Example News" || feed.Link != "https://example.org/" || feed.Description != "Example feed" {
		t.Fatalf("bad channel: %+v", feed)
	}
	if len(feed.Items) != 2 {
		t.Fatalf("want 2 items, got %d", len(feed.Items))
	}
	first := feed.Items[0]
	if first.Title != "First post" || first.Link != "https://example.org/posts/1" {
		t.Fatalf("bad first item: %+v", first)
	}
	if first.Description != "<p>Full body text</p>" {
		t.Fatalf("content:encoded should win over description, got %q", first.Description)
	}
	if want := time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC); !first.PubDate.Equal(want) {
		t.Fatalf("bad pubDate %v want %v", first.PubDate, want)
	}
	if len(first.Categories) != 2 || first.Categories[0] != "alpha" {
		t.Fatalf("bad categories: %v", first.Categories)
	}
	if first.GUID != "post-1" {
		t.Fatalf("guid should keep non-URL value: %q", first.GUID)
	}
	second := feed.Items[1]
	if second.Author == nil || second.Author.Name != "Jane Doe" {
		t.Fatalf("dc:creator should map to author: %+v", second.Author)
	}
	if second.GUID != "https://example.org/posts/2" {
		t.Fatalf("missing guid should fall back to link: %q", second.GUID)
	}
}

func TestParseAtom(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Atom Example</title>
  <subtitle>Sub</subtitle>
  <link href="https://atom.example.org/"/>
  <link rel="self" href="https://atom.example.org/feed"/>
  <entry>
    <title>Entry One</title>
    <link href="https://atom.example.org/e1"/>
    <id>urn:uuid:aaa</id>
    <updated>2026-04-01T12:00:00Z</updated>
    <published>2026-03-31T10:00:00Z</published>
    <summary>Short summary</summary>
    <author><name>Alice</name></author>
    <category term="tech"/>
  </entry>
</feed>`)

	feed, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if feed.Title != "Atom Example" || feed.Link != "https://atom.example.org/" {
		t.Fatalf("bad feed header: %+v", feed)
	}
	if len(feed.Items) != 1 {
		t.Fatalf("want 1 item, got %d", len(feed.Items))
	}
	item := feed.Items[0]
	if item.Title != "Entry One" || item.Link != "https://atom.example.org/e1" {
		t.Fatalf("bad entry: %+v", item)
	}
	if item.Description != "Short summary" {
		t.Fatalf("summary should map to description: %q", item.Description)
	}
	if item.Author == nil || item.Author.Name != "Alice" {
		t.Fatalf("author missing: %+v", item.Author)
	}
	if want := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC); !item.PubDate.Equal(want) {
		t.Fatalf("published should win over updated, got %v", item.PubDate)
	}
}

func TestParseRDF(t *testing.T) {
	data := []byte(`<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns="http://purl.org/rss/1.0/" xmlns:dc="http://purl.org/dc/elements/1.1/">
<channel rdf:about="https://rdf.example.org">
  <title>RDF Site</title>
  <link>https://rdf.example.org</link>
  <description>RDF feed</description>
</channel>
<item rdf:about="https://rdf.example.org/a">
  <title>Item A</title>
  <link>https://rdf.example.org/a</link>
  <dc:date>2026-05-01T00:00:00Z</dc:date>
</item>
</rdf:RDF>`)

	feed, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if feed.Title != "RDF Site" || len(feed.Items) != 1 || feed.Items[0].Title != "Item A" {
		t.Fatalf("bad RDF parse: %+v", feed)
	}
	if feed.Items[0].PubDate.IsZero() {
		t.Fatal("dc:date should be parsed")
	}
}

func TestParseEmpty(t *testing.T) {
	if _, err := Parse(nil); err == nil {
		t.Fatal("empty input should error")
	}
	if _, err := Parse([]byte("<html><body>nope</body></html>")); err == nil {
		t.Fatal("non-feed input should error")
	}
}

func TestEnclosureBecomesImage(t *testing.T) {
	data := []byte(`<rss><channel><title>E</title><item><title>x</title>
<enclosure url="https://img.example.org/pic.jpg" type="image/jpeg"/>
<description>d</description></item></channel></rss>`)
	feed, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	desc := feed.Items[0].Description
	if want := `<img src="https://img.example.org/pic.jpg"`; len(desc) < len(want) || desc[:len(want)] != want {
		t.Fatalf("enclosure image not prepended: %q", desc)
	}
}

var _ = models.Item{} // keep models import for clarity
