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
// If allowedTags is nil, all HTML is stripped (plain text only)
func SanitizeHTML(htmlStr string, allowedTags []string, allowedAttrs []string) (string, error) {
	if len(allowedTags) == 0 {
		// Strip all HTML, return plain text
		return stripHTML(htmlStr), nil
	}

	// Build allowed set
	allowed := make(map[string]bool)
	for _, tag := range allowedTags {
		allowed[tag] = true
	}

	// Parse and sanitize
	doc, err := htmlnode.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	var buf bytes.Buffer
	sanitizeNode(doc, &buf, allowed)
	return buf.String(), nil
}

// DefaultSanitize uses safe defaults (no tags allowed)
func DefaultSanitize(htmlStr string) (string, error) {
	return stripHTML(htmlStr), nil
}

// stripHTML removes all HTML tags and returns plain text
func stripHTML(htmlStr string) string {
	// Quick and dirty HTML tag removal
	re := regexp.MustCompile(`<[^>]*>`)
	cleaned := re.ReplaceAllString(htmlStr, "")

	// Decode entities
	cleaned = htmlentity.UnescapeString(cleaned)

	// Normalize whitespace
	return normalizeWhitespace(cleaned)
}

// sanitizeNode recursively sanitizes HTML nodes
func sanitizeNode(n *htmlnode.Node, buf *bytes.Buffer, allowed map[string]bool) {
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
			// Write attributes (could filter by allowedAttrs here)
			for _, attr := range n.Attr {
				buf.WriteString(fmt.Sprintf(` %s="%s"`, attr.Key, htmlentity.EscapeString(attr.Val)))
			}
			buf.WriteString(">")

			// Recursively process children
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				sanitizeNode(c, buf, allowed)
			}

			// Write closing tag
			buf.WriteString("</" + n.Data + ">")
		} else {
			// Tag not allowed, just process children
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				sanitizeNode(c, buf, allowed)
			}
		}
	case htmlnode.DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			sanitizeNode(c, buf, allowed)
		}
	}
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
	return doc.Html()
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

	return doc.Html()
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

	return doc.Html()
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

	return doc.Html()
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

	return doc.Html()
}

// CleanDescription applies full pipeline for content cleaning
func CleanDescription(htmlStr, baseURL string, opts CleanOptions) (string, error) {
	result := htmlStr

	// 1. Decode HTML entities
	result = DecodeHTMLEntities(result)

	// 2. Remove scripts
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
	// Remove all HTML tags
	text := stripHTML(htmlStr)

	// Normalize whitespace
	lines := strings.Split(text, "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
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

// Truncate truncates text to maxLength and adds suffix
func Truncate(text string, maxLength int, suffix string) string {
	if len(text) <= maxLength {
		return text
	}

	// Try to truncate at word boundary
	truncated := text[:maxLength]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}

	return truncated + suffix
}

// CollapseWhitespace normalizes whitespace in a string
func CollapseWhitespace(s string) string {
	return normalizeWhitespace(s)
}
