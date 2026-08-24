// Package rssfeed parses RSS 2.0, Atom and RSS 1.0 (RDF) documents into models.Feed,
// enabling thin "native feed" wrapper routes that normalize upstream feeds.
package rssfeed

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/xihale/syndi/pkg/models"
	dateutil "github.com/xihale/syndi/pkg/utils/date"
)

type rawItem struct {
	Title          string     `xml:"title"`
	Link           []rawLink  `xml:"link"`
	GUID           rawGUID    `xml:"guid"`
	Description    string     `xml:"description"`
	Summary        string     `xml:"summary"`
	ContentEncoded string     `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
	Content        string     `xml:"content"`
	PubDate        string     `xml:"pubDate"`
	DCDate         string     `xml:"http://purl.org/dc/elements/1.1/ date"`
	Published      string     `xml:"published"`
	Updated        string     `xml:"updated"`
	Author         rawPerson  `xml:"author"`
	DCCreator      string     `xml:"http://purl.org/dc/elements/1.1/ creator"`
	Category       []string   `xml:"category"`
	Enclosure      []rawEnclo `xml:"enclosure"`
}

type rawLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Text string `xml:",chardata"`
}

type rawGUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

type rawEnclo struct {
	URL    string `xml:"url,attr"`
	Type   string `xml:"type,attr"`
	Length string `xml:"length,attr"`
}

// rawPerson covers both RSS <author>addr</author> and Atom <author><name/></author>.
type rawPerson struct {
	Name  string `xml:"name"`
	Email string `xml:"email"`
	Raw   string `xml:",chardata"`
}

type rawChannel struct {
	Title       string    `xml:"title"`
	Link        []rawLink `xml:"link"`
	Description string    `xml:"description"`
	Subtitle    string    `xml:"subtitle"`
	Items       []rawItem `xml:"item"`
	Entries     []rawItem `xml:"entry"`
}

type rawDoc struct {
	XMLName xml.Name
	// Inline channel fields catch Atom roots (<feed><title/><entry/>...)
	// and RDF top-level items (<rdf:RDF><item/>...).
	rawChannel
	// Channel catches the <channel> child of <rss> and <rdf:RDF> documents.
	Channel rawChannel `xml:"channel"`
}

// Parse detects the feed format and converts it into a models.Feed.
// It accepts RSS 2.0, Atom and RSS 1.0 (RDF).
func Parse(data []byte) (*models.Feed, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, fmt.Errorf("rssfeed: empty document")
	}
	var doc rawDoc
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	dec.CharsetReader = charsetReader
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("rssfeed: parse xml: %w", err)
	}

	switch doc.XMLName.Local {
	case "rss":
		return buildFeed(doc.Channel, doc.Channel.Items), nil
	case "RDF":
		return buildFeed(doc.Channel, append(doc.Channel.Items, doc.rawChannel.Items...)), nil
	case "feed":
		return buildFeed(doc.rawChannel, doc.rawChannel.Entries), nil
	default:
		if len(doc.rawChannel.Entries) > 0 {
			return buildFeed(doc.rawChannel, doc.rawChannel.Entries), nil
		}
		if len(doc.Channel.Items) > 0 {
			return buildFeed(doc.Channel, doc.Channel.Items), nil
		}
		return nil, fmt.Errorf("rssfeed: unsupported feed root <%s>", doc.XMLName.Local)
	}
}

// ParseBytes parses raw feed bytes; convenience alias of Parse.
func ParseBytes(data []byte) (*models.Feed, error) { return Parse(data) }

func buildFeed(ch rawChannel, items []rawItem) *models.Feed {
	feed := &models.Feed{
		Title:       cleanText(firstNonEmpty(ch.Title)),
		Description: cleanText(firstNonEmpty(ch.Description, ch.Subtitle)),
		Link:        pickLink(ch.Link),
	}
	for _, ri := range items {
		item := convertItem(ri)
		if item == nil {
			continue
		}
		feed.Items = append(feed.Items, *item)
	}
	if feed.Items == nil {
		feed.Items = []models.Item{}
	}
	return feed
}

func convertItem(ri rawItem) *models.Item {
	title := cleanText(ri.Title)
	link := pickLink(ri.Link)
	description := firstNonEmpty(ri.ContentEncoded, ri.Content, ri.Description, ri.Summary)
	description = cleanHTML(description)

	pubRaw := firstNonEmpty(ri.PubDate, ri.Published, ri.DCDate, ri.Updated)
	var pub time.Time
	if pubRaw != "" {
		if t, err := dateutil.ParseDate(strings.TrimSpace(pubRaw)); err == nil {
			pub = t
		}
	}

	guid := strings.TrimSpace(ri.GUID.Value)
	if guid == "" {
		if link != "" {
			guid = link
		} else {
			guid = title
		}
	}

	if title == "" && link == "" && description == "" {
		return nil
	}

	item := &models.Item{
		Title:       title,
		Link:        link,
		Description: description,
		PubDate:     pub,
		GUID:        guid,
	}
	authorName := cleanText(firstNonEmpty(ri.DCCreator, ri.Author.Name, stripEmail(ri.Author.Raw), stripEmail(ri.Author.Email)))
	if authorName != "" {
		item.Author = &models.Author{Name: authorName}
	}
	for _, c := range ri.Category {
		c = cleanText(c)
		if c != "" {
			item.Categories = append(item.Categories, c)
		}
	}
	if enc := firstEnclosure(ri.Enclosure); enc != "" {
		img := fmt.Sprintf(`<img src="%s" alt="enclosure"/><br/>`, enc)
		item.Description = img + item.Description
	}
	return item
}

func pickLink(links []rawLink) string {
	// Prefer plain links / alternate links with href, then non-empty text.
	for _, l := range links {
		if l.Href != "" && (l.Rel == "" || l.Rel == "alternate") {
			return strings.TrimSpace(l.Href)
		}
	}
	for _, l := range links {
		if t := strings.TrimSpace(l.Text); t != "" && strings.HasPrefix(t, "http") {
			return t
		}
		if l.Href != "" {
			return strings.TrimSpace(l.Href)
		}
	}
	return ""
}

func firstEnclosure(encs []rawEnclo) string {
	for _, e := range encs {
		if e.URL != "" {
			return e.URL
		}
	}
	return ""
}

func stripEmail(author string) string {
	author = strings.TrimSpace(author)
	// RFC 822 style: "Name <email@example.com>"
	if i := strings.Index(author, "<"); i > 0 {
		return strings.TrimSpace(strings.TrimSuffix(author[:i], "("))
	}
	if strings.Count(author, "@") == 1 && !strings.ContainsAny(author, " \t\n/") {
		return author // bare address used as name
	}
	return author
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func cleanText(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// cleanHTML normalizes whitespace outside of tags so HTML payloads keep their markup.
func cleanHTML(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var out strings.Builder
	inTag := false
	lastSpace := false
	reader := strings.NewReader(s)
	buf := make([]byte, 1)
	for {
		n, err := reader.Read(buf)
		if n == 0 {
			if err == io.EOF {
				break
			}
			continue
		}
		ch := buf[0]
		if ch == '<' {
			inTag = true
			out.WriteByte(ch)
			lastSpace = false
			continue
		}
		if ch == '>' {
			inTag = false
			out.WriteByte(ch)
			lastSpace = false
			continue
		}
		if inTag {
			out.WriteByte(ch)
			continue
		}
		if ch == '\n' || ch == '\r' || ch == '\t' || ch == ' ' {
			if !lastSpace {
				out.WriteByte(' ')
				lastSpace = true
			}
			continue
		}
		out.WriteByte(ch)
		lastSpace = false
	}
	return strings.TrimSpace(out.String())
}

func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	charset = strings.ToLower(strings.TrimSpace(charset))
	switch charset {
	case "", "utf-8", "utf8", "us-ascii", "ascii":
		return input, nil
	case "gbk", "gb2312", "gb18030", "big5", "shift_jis", "sjis", "euc-jp", "euc-kr", "windows-1252", "latin1", "iso-8859-1", "iso8859-1", "koi8-r":
		// Best-effort: treat as UTF-8 passthrough. Most modern feeds are UTF-8;
		// strict decoding is disabled above so invalid bytes become replacement runes
		// rather than hard errors.
		return input, nil
	default:
		return input, nil
	}
}
