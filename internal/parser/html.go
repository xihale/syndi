package parser

import (
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Document wraps goquery.Document with helper methods
type Document struct {
	*goquery.Document
}

// Load parses HTML from a reader
func Load(r io.Reader) (*Document, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}
	return &Document{Document: doc}, nil
}

// LoadString parses HTML from a string
func LoadString(s string) (*Document, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(s))
	if err != nil {
		return nil, err
	}
	return &Document{Document: doc}, nil
}

// Text returns the text content of the first matching element
func (d *Document) Text(selector string) string {
	return d.Find(selector).First().Text()
}

// TextAll returns concatenated text content of all matching elements
func (d *Document) TextAll(selector string) string {
	var texts []string
	d.Find(selector).Each(func(i int, s *goquery.Selection) {
		texts = append(texts, s.Text())
	})
	return strings.Join(texts, " ")
}

// Attr returns the attribute value of the first matching element
func (d *Document) Attr(selector, attrName string) (string, bool) {
	val, exists := d.Find(selector).First().Attr(attrName)
	return val, exists
}

// AttrAll returns all attribute values for matching elements
func (d *Document) AttrAll(selector, attrName string) []string {
	var vals []string
	d.Find(selector).Each(func(i int, s *goquery.Selection) {
		if val, ok := s.Attr(attrName); ok {
			vals = append(vals, val)
		}
	})
	return vals
}

// Each iterates over matching elements
func (d *Document) Each(selector string, fn func(int, *Selection)) {
	d.Find(selector).Each(func(i int, s *goquery.Selection) {
		fn(i, &Selection{Selection: s})
	})
}

// Map maps over matching elements
func (d *Document) Map(selector string, fn func(int, *Selection) string) []string {
	var result []string
	d.Find(selector).Each(func(i int, s *goquery.Selection) {
		if str := fn(i, &Selection{Selection: s}); str != "" {
			result = append(result, str)
		}
	})
	return result
}

// First returns the first matching element
func (d *Document) First(selector string) *Selection {
	s := d.Find(selector).First()
	if s.Length() == 0 {
		return nil
	}
	return &Selection{Selection: s}
}

// Find returns a new Selection for matching elements
func (d *Document) FindSelector(selector string) *Selection {
	s := d.Find(selector)
	if s.Length() == 0 {
		return nil
	}
	return &Selection{Selection: s}
}

// Selection wraps goquery.Selection with helper methods
type Selection struct {
	*goquery.Selection
}

// Text returns the text content
func (s *Selection) Text() string {
	if s == nil || s.Selection == nil {
		return ""
	}
	return s.Selection.Text()
}

// TextTrim returns trimmed text content
func (s *Selection) TextTrim() string {
	return strings.TrimSpace(s.Text())
}

// Attr returns an attribute value
func (s *Selection) Attr(name string) (string, bool) {
	if s == nil || s.Selection == nil {
		return "", false
	}
	return s.Selection.Attr(name)
}

// AttrOr returns an attribute value or default
func (s *Selection) AttrOr(name, defaultValue string) string {
	if val, ok := s.Attr(name); ok {
		return val
	}
	return defaultValue
}

// Find finds descendant elements
func (s *Selection) Find(selector string) *Selection {
	if s == nil || s.Selection == nil {
		return nil
	}
	sel := s.Selection.Find(selector)
	if sel.Length() == 0 {
		return nil
	}
	return &Selection{Selection: sel}
}

// Children returns children elements
func (s *Selection) Children() *Selection {
	if s == nil || s.Selection == nil {
		return nil
	}
	sel := s.Selection.Children()
	if sel.Length() == 0 {
		return nil
	}
	return &Selection{Selection: sel}
}

// Parent returns the parent element
func (s *Selection) Parent() *Selection {
	if s == nil || s.Selection == nil {
		return nil
	}
	sel := s.Selection.Parent()
	if sel.Length() == 0 {
		return nil
	}
	return &Selection{Selection: sel}
}

// Each iterates over elements in the selection
func (s *Selection) Each(fn func(int, *Selection)) {
	if s == nil || s.Selection == nil {
		return
	}
	s.Selection.Each(func(i int, sel *goquery.Selection) {
		fn(i, &Selection{Selection: sel})
	})
}

// Length returns the number of elements
func (s *Selection) Length() int {
	if s == nil || s.Selection == nil {
		return 0
	}
	return s.Selection.Length()
}

// Is checks if the selection matches a selector
func (s *Selection) Is(selector string) bool {
	if s == nil || s.Selection == nil {
		return false
	}
	return s.Selection.Is(selector)
}

// HasClass checks if the element has a class
func (s *Selection) HasClass(class string) bool {
	if s == nil || s.Selection == nil {
		return false
	}
	return s.Selection.HasClass(class)
}

// Filter filters elements
func (s *Selection) Filter(selector string) *Selection {
	if s == nil || s.Selection == nil {
		return nil
	}
	sel := s.Selection.Filter(selector)
	if sel.Length() == 0 {
		return nil
	}
	return &Selection{Selection: sel}
}

// Next returns the next sibling element
func (s *Selection) Next() *Selection {
	if s == nil || s.Selection == nil {
		return nil
	}
	sel := s.Selection.Next()
	if sel.Length() == 0 {
		return nil
	}
	return &Selection{Selection: sel}
}

// Prev returns the previous sibling element
func (s *Selection) Prev() *Selection {
	if s == nil || s.Selection == nil {
		return nil
	}
	sel := s.Selection.Prev()
	if sel.Length() == 0 {
		return nil
	}
	return &Selection{Selection: sel}
}

// NextAll returns all following siblings
func (s *Selection) NextAll() *Selection {
	if s == nil || s.Selection == nil {
		return nil
	}
	sel := s.Selection.NextAll()
	if sel.Length() == 0 {
		return nil
	}
	return &Selection{Selection: sel}
}

// PrevAll returns all preceding siblings
func (s *Selection) PrevAll() *Selection {
	if s == nil || s.Selection == nil {
		return nil
	}
	sel := s.Selection.PrevAll()
	if sel.Length() == 0 {
		return nil
	}
	return &Selection{Selection: sel}
}

// Parents returns all ancestor elements
func (s *Selection) Parents() *Selection {
	if s == nil || s.Selection == nil {
		return nil
	}
	sel := s.Selection.Parents()
	if sel.Length() == 0 {
		return nil
	}
	return &Selection{Selection: sel}
}

// Closest returns the closest ancestor matching the selector
func (s *Selection) Closest(selector string) *Selection {
	if s == nil || s.Selection == nil {
		return nil
	}
	sel := s.Selection.Closest(selector)
	if sel.Length() == 0 {
		return nil
	}
	return &Selection{Selection: sel}
}

// Html returns the inner HTML of the first element
func (s *Selection) Html() (string, error) {
	if s == nil || s.Selection == nil {
		return "", nil
	}
	return s.Selection.Html()
}

// OuterHtml returns the outer HTML of the first element
func (s *Selection) OuterHtml() (string, error) {
	if s == nil || s.Selection == nil {
		return "", nil
	}
	return goquery.OuterHtml(s.Selection)
}

// Eq returns the element at the specified index
func (s *Selection) Eq(index int) *Selection {
	if s == nil || s.Selection == nil {
		return nil
	}
	sel := s.Selection.Eq(index)
	if sel.Length() == 0 {
		return nil
	}
	return &Selection{Selection: sel}
}

// First returns the first element in the selection
func (s *Selection) First() *Selection {
	if s == nil || s.Selection == nil {
		return nil
	}
	sel := s.Selection.First()
	if sel.Length() == 0 {
		return nil
	}
	return &Selection{Selection: sel}
}

// Last returns the last element in the selection
func (s *Selection) Last() *Selection {
	if s == nil || s.Selection == nil {
		return nil
	}
	sel := s.Selection.Last()
	if sel.Length() == 0 {
		return nil
	}
	return &Selection{Selection: sel}
}

// Slice returns a subset of elements
func (s *Selection) Slice(start, end int) *Selection {
	if s == nil || s.Selection == nil {
		return nil
	}
	sel := s.Selection.Slice(start, end)
	if sel.Length() == 0 {
		return nil
	}
	return &Selection{Selection: sel}
}
