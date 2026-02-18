package parser

import (
	"strings"
	"testing"
)

func TestLoadString(t *testing.T) {
	html := `<html><body><h1>Hello</h1></body></html>`

	doc, err := LoadString(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc == nil {
		t.Fatal("expected non-nil document")
	}
}

func TestDocument_Text(t *testing.T) {
	html := `<html><body><h1>Hello World</h1></body></html>`

	doc, _ := LoadString(html)
	text := doc.Text("h1")

	if text != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", text)
	}
}

func TestDocument_Text_NotFound(t *testing.T) {
	html := `<html><body><p>Content</p></body></html>`

	doc, _ := LoadString(html)
	text := doc.Text("h1")

	if text != "" {
		t.Errorf("expected empty string, got '%s'", text)
	}
}

func TestDocument_TextAll(t *testing.T) {
	html := `<html><body><p>First</p><p>Second</p></body></html>`

	doc, _ := LoadString(html)
	text := doc.TextAll("p")

	expected := "First Second"
	if text != expected {
		t.Errorf("expected '%s', got '%s'", expected, text)
	}
}

func TestDocument_Attr(t *testing.T) {
	html := `<html><body><a href="https://example.com">Link</a></body></html>`

	doc, _ := LoadString(html)
	href, ok := doc.Attr("a", "href")

	if !ok {
		t.Error("expected attribute to exist")
	}

	if href != "https://example.com" {
		t.Errorf("expected 'https://example.com', got '%s'", href)
	}
}

func TestDocument_Attr_NotFound(t *testing.T) {
	html := `<html><body><span>Text</span></body></html>`

	doc, _ := LoadString(html)
	_, ok := doc.Attr("span", "href")

	if ok {
		t.Error("expected attribute to not exist")
	}
}

func TestDocument_AttrAll(t *testing.T) {
	html := `<html><body>
		<a href="/link1">Link 1</a>
		<a href="/link2">Link 2</a>
	</body></html>`

	doc, _ := LoadString(html)
	hrefs := doc.AttrAll("a", "href")

	if len(hrefs) != 2 {
		t.Errorf("expected 2 hrefs, got %d", len(hrefs))
	}

	if hrefs[0] != "/link1" {
		t.Errorf("expected '/link1', got '%s'", hrefs[0])
	}

	if hrefs[1] != "/link2" {
		t.Errorf("expected '/link2', got '%s'", hrefs[1])
	}
}

func TestDocument_First(t *testing.T) {
	html := `<html><body><p class="test">First</p><p class="test">Second</p></body></html>`

	doc, _ := LoadString(html)
	first := doc.First("p.test")

	if first == nil {
		t.Fatal("expected non-nil selection")
	}

	if first.Text() != "First" {
		t.Errorf("expected 'First', got '%s'", first.Text())
	}
}

func TestDocument_First_NotFound(t *testing.T) {
	html := `<html><body><p>Content</p></body></html>`

	doc, _ := LoadString(html)
	first := doc.First("h1")

	if first != nil {
		t.Error("expected nil selection")
	}
}

func TestDocument_Each(t *testing.T) {
	html := `<html><body><p>1</p><p>2</p><p>3</p></body></html>`

	doc, _ := LoadString(html)
	count := 0

	doc.Each("p", func(i int, s *Selection) {
		count++
	})

	if count != 3 {
		t.Errorf("expected 3 iterations, got %d", count)
	}
}

func TestDocument_Map(t *testing.T) {
	html := `<html><body><p>A</p><p>B</p><p>C</p></body></html>`

	doc, _ := LoadString(html)
	result := doc.Map("p", func(i int, s *Selection) string {
		return strings.ToLower(s.Text())
	})

	expected := []string{"a", "b", "c"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(result))
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("item %d: expected '%s', got '%s'", i, expected[i], v)
		}
	}
}

func TestSelection_Text(t *testing.T) {
	html := `<html><body><p>Hello World</p></body></html>`

	doc, _ := LoadString(html)
	sel := doc.First("p")

	if sel.Text() != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", sel.Text())
	}
}

func TestSelection_TextTrim(t *testing.T) {
	html := `<html><body><p>   Hello World   </p></body></html>`

	doc, _ := LoadString(html)
	sel := doc.First("p")

	if sel.TextTrim() != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", sel.TextTrim())
	}
}

func TestSelection_Attr(t *testing.T) {
	html := `<html><body><a href="https://example.com" title="Test">Link</a></body></html>`

	doc, _ := LoadString(html)
	sel := doc.First("a")

	href, ok := sel.Attr("href")
	if !ok {
		t.Error("expected href to exist")
	}
	if href != "https://example.com" {
		t.Errorf("expected 'https://example.com', got '%s'", href)
	}

	title, ok := sel.Attr("title")
	if !ok {
		t.Error("expected title to exist")
	}
	if title != "Test" {
		t.Errorf("expected 'Test', got '%s'", title)
	}
}

func TestSelection_AttrOr(t *testing.T) {
	html := `<html><body><a href="https://example.com">Link</a></body></html>`

	doc, _ := LoadString(html)
	sel := doc.First("a")

	// Existing attribute
	href := sel.AttrOr("href", "default")
	if href != "https://example.com" {
		t.Errorf("expected 'https://example.com', got '%s'", href)
	}

	// Non-existing attribute
	title := sel.AttrOr("title", "default")
	if title != "default" {
		t.Errorf("expected 'default', got '%s'", title)
	}
}

func TestSelection_Find(t *testing.T) {
	html := `<html><body><div class="container"><p>Found</p></div></body></html>`

	doc, _ := LoadString(html)
	sel := doc.First(".container")

	found := sel.Find("p")
	if found == nil {
		t.Fatal("expected non-nil selection")
	}

	if found.Text() != "Found" {
		t.Errorf("expected 'Found', got '%s'", found.Text())
	}
}

func TestSelection_Find_NotFound(t *testing.T) {
	html := `<html><body><div class="container">Content</div></body></html>`

	doc, _ := LoadString(html)
	sel := doc.First(".container")

	found := sel.Find("p")
	if found != nil {
		t.Error("expected nil selection")
	}
}

func TestSelection_Next(t *testing.T) {
	html := `<html><body><p class="a">A</p><p class="b">B</p><p class="c">C</p></body></html>`

	doc, _ := LoadString(html)
	sel := doc.First(".a")

	next := sel.Next()
	if next == nil {
		t.Fatal("expected non-nil selection")
	}

	if next.Text() != "B" {
		t.Errorf("expected 'B', got '%s'", next.Text())
	}
}

func TestSelection_Prev(t *testing.T) {
	html := `<html><body><p class="a">A</p><p class="b">B</p><p class="c">C</p></body></html>`

	doc, _ := LoadString(html)
	sel := doc.First(".c")

	prev := sel.Prev()
	if prev == nil {
		t.Fatal("expected non-nil selection")
	}

	if prev.Text() != "B" {
		t.Errorf("expected 'B', got '%s'", prev.Text())
	}
}

func TestSelection_Parent(t *testing.T) {
	html := `<html><body><div><p>Child</p></div></body></html>`

	doc, _ := LoadString(html)
	sel := doc.First("p")

	parent := sel.Parent()
	if parent == nil {
		t.Fatal("expected non-nil selection")
	}

	if !parent.Is("div") {
		t.Error("expected parent to be div")
	}
}

func TestSelection_Children(t *testing.T) {
	html := `<html><body><ul><li>A</li><li>B</li></ul></body></html>`

	doc, _ := LoadString(html)
	sel := doc.First("ul")

	children := sel.Children()
	if children == nil {
		t.Fatal("expected non-nil selection")
	}

	if children.Length() != 2 {
		t.Errorf("expected 2 children, got %d", children.Length())
	}
}

func TestSelection_HasClass(t *testing.T) {
	html := `<html><body><p class="foo bar baz">Text</p></body></html>`

	doc, _ := LoadString(html)
	sel := doc.First("p")

	if !sel.HasClass("foo") {
		t.Error("expected element to have class 'foo'")
	}

	if !sel.HasClass("bar") {
		t.Error("expected element to have class 'bar'")
	}

	if sel.HasClass("qux") {
		t.Error("expected element to not have class 'qux'")
	}
}

func TestSelection_Is(t *testing.T) {
	html := `<html><body><p class="test">Text</p></body></html>`

	doc, _ := LoadString(html)
	sel := doc.First("p")

	if !sel.Is(".test") {
		t.Error("expected element to match selector")
	}

	if sel.Is("div") {
		t.Error("expected element to not match selector")
	}
}

func TestSelection_Length(t *testing.T) {
	html := `<html><body><p>A</p><p>B</p><p>C</p></body></html>`

	doc, _ := LoadString(html)
	sel := doc.FindSelector("p")

	if sel == nil {
		t.Fatal("expected non-nil selection")
	}

	if sel.Length() != 3 {
		t.Errorf("expected length 3, got %d", sel.Length())
	}
}

func TestSelection_Eq(t *testing.T) {
	html := `<html><body><p>A</p><p>B</p><p>C</p></body></html>`

	doc, _ := LoadString(html)
	sel := doc.FindSelector("p")

	if sel == nil {
		t.Fatal("expected non-nil selection")
	}

	second := sel.Eq(1)
	if second == nil {
		t.Fatal("expected non-nil selection")
	}

	if second.Text() != "B" {
		t.Errorf("expected 'B', got '%s'", second.Text())
	}
}

func TestSelection_FirstLast(t *testing.T) {
	html := `<html><body><p>A</p><p>B</p><p>C</p></body></html>`

	doc, _ := LoadString(html)
	sel := doc.FindSelector("p")

	if sel == nil {
		t.Fatal("expected non-nil selection")
	}

	first := sel.First()
	if first.Text() != "A" {
		t.Errorf("expected 'A', got '%s'", first.Text())
	}

	last := sel.Last()
	if last.Text() != "C" {
		t.Errorf("expected 'C', got '%s'", last.Text())
	}
}

func TestSelection_Slice(t *testing.T) {
	html := `<html><body><p>A</p><p>B</p><p>C</p><p>D</p><p>E</p></body></html>`

	doc, _ := LoadString(html)
	sel := doc.FindSelector("p")

	if sel == nil {
		t.Fatal("expected non-nil selection")
	}

	sliced := sel.Slice(1, 3)
	if sliced == nil {
		t.Fatal("expected non-nil selection")
	}

	if sliced.Length() != 2 {
		t.Errorf("expected length 2, got %d", sliced.Length())
	}

	texts := []string{sliced.Eq(0).Text(), sliced.Eq(1).Text()}
	if texts[0] != "B" || texts[1] != "C" {
		t.Errorf("expected ['B', 'C'], got %v", texts)
	}
}

func TestSelection_Nil(t *testing.T) {
	var sel *Selection

	if sel.Text() != "" {
		t.Error("expected empty text for nil selection")
	}

	if sel.TextTrim() != "" {
		t.Error("expected empty trimmed text for nil selection")
	}

	if _, ok := sel.Attr("href"); ok {
		t.Error("expected false for nil selection attribute")
	}

	if sel.AttrOr("href", "default") != "default" {
		t.Error("expected default value for nil selection")
	}

	if sel.Length() != 0 {
		t.Error("expected length 0 for nil selection")
	}
}
