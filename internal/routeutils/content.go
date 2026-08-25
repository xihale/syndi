package routeutils

import (
	"bytes"
	"fmt"
	htmlentity "html"
	"regexp"
	"strings"
	"unicode"

	"github.com/xihale/syndi/internal/parser"
	htmlnode "golang.org/x/net/html"
)

// CleanOptions contains options for CleanDescription
type CleanOptions struct {
	Sanitize       bool
	Suffix         string
	BaseURL        string
	ReferrerPolicy bool
	RemoveScripts  bool
	FixLazyImages  bool
	ResolveLinks   bool
	AllowedTags    []string
	AllowedAttrs   []string
}

// DefaultCleanOptions returns sensible defaults for content cleaning
func DefaultCleanOptions() CleanOptions {
	return CleanOptions{
		Sanitize:       false,
		Suffix:         "",
		BaseURL:        "",
		ReferrerPolicy: true,
		RemoveScripts:  true,
		FixLazyImages:  true,
		ResolveLinks:   true,
		AllowedTags:    nil, // nil means no sanitization
		AllowedAttrs:   nil,
	}
}

// DecodeHTMLEntities decodes HTML entities like &amp; &lt; &gt;
func DecodeHTMLEntities(s string) string {
	return htmlentity.UnescapeString(s)
}

// SanitizeHTML removes dangerous tags/attributes (XSS protection)
// If allowedTags is nil, all HTML is stripped (plain text only).
// allowedAttrs whitelists element attributes (nil selects a small safe
// default); URL-valued attributes additionally reject script-capable schemes.
func SanitizeHTML(htmlStr string, allowedTags []string, allowedAttrs []string) (string, error) {
	if len(allowedTags) == 0 {
		// Strip all HTML, return plain text
		return stripHTML(htmlStr), nil
	}

	// Build allowed sets
	allowed := make(map[string]bool)
	for _, tag := range allowedTags {
		allowed[tag] = true
	}
	attrs := make(map[string]bool)
	if len(allowedAttrs) == 0 {
		allowedAttrs = defaultAllowedAttrs
	}
	for _, attr := range allowedAttrs {
		attrs[strings.ToLower(attr)] = true
	}

	// Parse and sanitize
	doc, err := htmlnode.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	var buf bytes.Buffer
	sanitizeNode(doc, &buf, allowed, attrs)
	return buf.String(), nil
}

// defaultAllowedAttrs is used when AllowedAttrs is not provided: enough to
// keep links/images working while dropping event handlers and styling.
var defaultAllowedAttrs = []string{"href", "src", "alt", "title", "width", "height"}

// sanitizeURL reports whether a URL-valued attribute value is safe.
func sanitizeURL(u string) bool {
	u = strings.TrimSpace(strings.ToLower(u))
	if i := strings.IndexAny(u, "?#"); i >= 0 {
		u = u[:i] // scheme must appear before any query/fragment
	}
	for _, prefix := range []string{"javascript:", "vbscript:", "data:"} {
		if strings.HasPrefix(u, prefix) {
			return false
		}
	}
	return true
}

// DefaultSanitize uses safe defaults (no tags allowed)
func DefaultSanitize(htmlStr string) (string, error) {
	return stripHTML(htmlStr), nil
}

// stripHTML removes all HTML tags and returns plain text
func stripHTML(htmlStr string) string {
	// Drop script/style blocks including their text content first, so the
	// tag-only removal below cannot leak raw JS/CSS into "plain text".
	scriptRe := regexp.MustCompile(`(?is)<script\b[^>]*>.*?(?:</script\s*>|\z)`)
	styleRe := regexp.MustCompile(`(?is)<style\b[^>]*>.*?(?:</style\s*>|\z)`)
	cleaned := scriptRe.ReplaceAllString(htmlStr, "")
	cleaned = styleRe.ReplaceAllString(cleaned, "")

	// Quick and dirty HTML tag removal
	re := regexp.MustCompile(`<[^>]*>`)
	cleaned = re.ReplaceAllString(cleaned, "")

	// Decode entities
	cleaned = htmlentity.UnescapeString(cleaned)

	// Normalize whitespace
	return normalizeWhitespace(cleaned)
}

// sanitizeNode recursively sanitizes HTML nodes
func sanitizeNode(n *htmlnode.Node, buf *bytes.Buffer, allowed map[string]bool, allowedAttrs map[string]bool) {
	if n == nil {
		return
	}

	switch n.Type {
	case htmlnode.TextNode:
		buf.WriteString(n.Data)
	case htmlnode.ElementNode:
		if allowed[n.Data] {
			// Write opening tag
			buf.WriteString("<" + n.Data)
			// Write attributes, filtered by the whitelist; URL-valued
			// attributes additionally reject script-capable schemes.
			for _, attr := range n.Attr {
				key := strings.ToLower(attr.Key)
				if !allowedAttrs[key] {
					continue
				}
				if (key == "href" || key == "src") && !sanitizeURL(attr.Val) {
					continue
				}
				fmt.Fprintf(buf, ` %s="%s"`, attr.Key, htmlentity.EscapeString(attr.Val))
			}
			buf.WriteString(">")

			// Recursively process children
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				sanitizeNode(c, buf, allowed, allowedAttrs)
			}

			// Write closing tag
			buf.WriteString("</" + n.Data + ">")
		} else {
			// Tag not allowed, just process children
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				sanitizeNode(c, buf, allowed, allowedAttrs)
			}
		}
	case htmlnode.DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			sanitizeNode(c, buf, allowed, allowedAttrs)
		}
	}
}

// fragmentHTML serializes only the body contents of a parsed document, so
// fragments passed through the pipeline don't gain <html><head><body>
// wrappers (goquery.Html() on the document node emits the whole document).
func fragmentHTML(doc *parser.Document) (string, error) {
	if body := doc.Find("body"); body.Length() > 0 {
		return body.Html()
	}
	return doc.Html()
}

// FixLazyImages replaces data-src, data-original with src attribute
func FixLazyImages(htmlStr, baseURL string) (string, error) {
	doc, err := parser.LoadString(htmlStr)
	if err != nil {
		return "", err
	}

	// Find all img elements and fix lazy loading attributes
	attributes := []string{"data-src", "data-original", "data-lazy-src", "data-actualsrc"}

	for _, attr := range attributes {
		doc.Each(fmt.Sprintf("img[%s]", attr), func(i int, sel *parser.Selection) {
			if src, ok := sel.Attr(attr); ok && src != "" {
				sel.SetAttr("src", src)
			}
		})
	}

	// Return the modified HTML
	return fragmentHTML(doc)
}

// RemoveScripts removes all <script> tags completely
func RemoveScripts(htmlStr string) (string, error) {
	doc, err := parser.LoadString(htmlStr)
	if err != nil {
		return "", err
	}

	// Remove all script tags
	doc.Each("script", func(i int, sel *parser.Selection) {
		sel.Remove()
	})

	// Remove style tags
	doc.Each("style", func(i int, sel *parser.Selection) {
		sel.Remove()
	})

	return fragmentHTML(doc)
}

// AddReferrerPolicy adds referrerpolicy="no-referrer" to links, images, iframes
func AddReferrerPolicy(htmlStr string) (string, error) {
	doc, err := parser.LoadString(htmlStr)
	if err != nil {
		return "", err
	}

	// Add referrer policy to links, images, iframes, videos, audio
	tags := []string{"a", "img", "iframe", "video", "audio", "source"}
	for _, tag := range tags {
		doc.Each(tag, func(i int, sel *parser.Selection) {
			sel.SetAttr("referrerpolicy", "no-referrer")
		})
	}

	return fragmentHTML(doc)
}

// ResolveLinksInHTML finds all relative links and makes them absolute
func ResolveLinksInHTML(htmlStr, baseURL string) (string, error) {
	doc, err := parser.LoadString(htmlStr)
	if err != nil {
		return "", err
	}

	// Resolve href attributes in <a> tags
	doc.Each("a[href]", func(i int, sel *parser.Selection) {
		if href, ok := sel.Attr("href"); ok {
			if absolute, err := ResolveURL(baseURL, href); err == nil {
				sel.SetAttr("href", absolute)
			}
		}
	})

	// Resolve src attributes
	doc.Each("[src]", func(i int, sel *parser.Selection) {
		if src, ok := sel.Attr("src"); ok {
			if absolute, err := ResolveURL(baseURL, src); err == nil {
				sel.SetAttr("src", absolute)
			}
		}
	})

	return fragmentHTML(doc)
}

// ResolveImages resolves src attributes in <img>, <video>, <source>, <iframe>
func ResolveImages(htmlStr, baseURL string) (string, error) {
	doc, err := parser.LoadString(htmlStr)
	if err != nil {
		return "", err
	}

	// Tags with src attributes
	tags := []string{"img", "video", "audio", "source", "iframe", "embed"}
	for _, tag := range tags {
		sel := fmt.Sprintf("%s[src]", tag)
		doc.Each(sel, func(i int, s *parser.Selection) {
			if src, ok := s.Attr("src"); ok {
				if absolute, err := ResolveURL(baseURL, src); err == nil {
					s.SetAttr("src", absolute)
				}
			}
		})
	}

	// Also handle poster attribute on video
	doc.Each("video[poster]", func(i int, sel *parser.Selection) {
		if poster, ok := sel.Attr("poster"); ok {
			if absolute, err := ResolveURL(baseURL, poster); err == nil {
				sel.SetAttr("poster", absolute)
			}
		}
	})

	return fragmentHTML(doc)
}

// CleanDescription applies full pipeline for content cleaning
func CleanDescription(htmlStr, baseURL string, opts CleanOptions) (string, error) {
	result := htmlStr

	// Note: deliberately no entity decoding up front. Decoding before the
	// HTML passes below would turn escaped text like &lt;img onerror=…&gt;
	// into live tags. Entities are only decoded on plain-text output paths
	// (stripHTML).

	// 1. Remove scripts
	if opts.RemoveScripts {
		var err error
		result, err = RemoveScripts(result)
		if err != nil {
			return "", fmt.Errorf("failed to remove scripts: %w", err)
		}
	}

	// 3. Fix lazy images
	if opts.FixLazyImages {
		var err error
		result, err = FixLazyImages(result, baseURL)
		if err != nil {
			return "", fmt.Errorf("failed to fix lazy images: %w", err)
		}
	}

	// 4. Resolve all links
	if opts.ResolveLinks && baseURL != "" {
		var err error
		result, err = ResolveLinksInHTML(result, baseURL)
		if err != nil {
			return "", fmt.Errorf("failed to resolve links: %w", err)
		}
	}

	// 5. Add referrer policy
	if opts.ReferrerPolicy {
		var err error
		result, err = AddReferrerPolicy(result)
		if err != nil {
			return "", fmt.Errorf("failed to add referrer policy: %w", err)
		}
	}

	// 6. Sanitize if requested
	if opts.Sanitize {
		var err error
		result, err = SanitizeHTML(result, opts.AllowedTags, opts.AllowedAttrs)
		if err != nil {
			return "", fmt.Errorf("failed to sanitize HTML: %w", err)
		}
	}

	// 7. Append suffix if provided
	if opts.Suffix != "" {
		result = result + opts.Suffix
	}

	return result, nil
}

// ExtractText strips HTML and returns plain text with normalized whitespace
func ExtractText(htmlStr string) string {
	// stripHTML already removes script/style content, strips tags, decodes
	// entities and collapses whitespace.
	return stripHTML(htmlStr)
}

// normalizeWhitespace collapses multiple whitespace characters into single spaces
func normalizeWhitespace(s string) string {
	var buf strings.Builder
	buf.Grow(len(s))

	inSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !inSpace && buf.Len() > 0 {
				buf.WriteRune(' ')
				inSpace = true
			}
		} else {
			buf.WriteRune(r)
			inSpace = false
		}
	}

	return strings.TrimSpace(buf.String())
}

// Truncate truncates text to maxLength runes and adds suffix.
// maxLength counts runes, not bytes (byte slicing would split multi-byte
// characters); negative maxLength returns the text unchanged.
func Truncate(text string, maxLength int, suffix string) string {
	if maxLength < 0 {
		return text
	}
	runes := []rune(text)
	if len(runes) <= maxLength {
		return text
	}

	// Try to truncate at word boundary
	truncated := string(runes[:maxLength])
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}

	return truncated + suffix
}

// CollapseWhitespace normalizes whitespace in a string
func CollapseWhitespace(s string) string {
	return normalizeWhitespace(s)
}
