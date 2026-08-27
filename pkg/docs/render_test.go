package docs

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

func renderToString(t *testing.T, tmpl *template.Template, data any) string {
	t.Helper()
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("render error: %v", err)
	}
	return buf.String()
}

func TestDocTemplatesRenderIndex(t *testing.T) {
	tmpls, err := parseDocTemplates()
	if err != nil {
		t.Fatalf("parse doc templates: %v", err)
	}

	out := renderToString(t, tmpls.index, PageData{Title: "t"})
	for _, want := range []string{
		"<title>t — Syndi</title>",
		`rel="stylesheet" href="/assets/css/docs.css"`,
		`type="module" src="/assets/js/home.js"`,
		`<input type="text" id="q"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("index page missing %q", want)
		}
	}
}

func TestDocTemplatesRenderRoute(t *testing.T) {
	tmpls, err := parseDocTemplates()
	if err != nil {
		t.Fatalf("parse doc templates: %v", err)
	}

	data := RoutePageData{
		Title:   "知乎热榜",
		Route: &RouteDoc{
			Path:        "/zhihu/hot",
			Name:        "知乎热榜",
			Description: "d",
			ExampleURL:  "/rss/zhihu/hot",
			CurlExample: "curl x",
			Categories:  []string{"social-media"},
		},
		Related: []*RouteDoc{},
		RouteEnvStatuses: []CredStatus{{
			Key:    "ZHIHU_COOKIES",
			Fields: []CredField{{Name: "z_c0", Present: false}},
		}},
	}

	out := renderToString(t, tmpls.route, data)
	for _, want := range []string{
		"<title>知乎热榜 — Syndi</title>",
		`<div class="d-path">/zhihu/hot</div>`,
		`<dd class="st no">missing</dd>`,
		`<form class="try-row">`,
		`type="module" src="/assets/js/detail.js"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("route page missing %q", want)
		}
	}

	data.RouteEnvStatuses[0].Fields[0].Present = true
	out = renderToString(t, tmpls.route, data)
	if !strings.Contains(out, `<dd class="st yes">ok</dd>`) || strings.Contains(out, "missing") {
		t.Error("configured credential field should render ok, not missing")
	}
}

func TestMatchPattern(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"/zhihu/hot", "/zhihu/hot", true},
		{"/zhihu/hot", "/zhihu/hot/x", false},
		{"/zhihu/hot", "/zhihu/cold", false},
		{"/bilibili/user/video/:uid", "/bilibili/user/video/208259", true},
		{"/bilibili/user/video/:uid", "/bilibili/user/video/208259/extra", false},
		{"/bilibili/user/video/:uid", "/bilibili/user/video", false},
		// A trailing wildcard also matches when the concrete path stops there,
		// mirroring gin's catch-all semantics used by the docs matcher.
		{"/telegram/route/*path", "/telegram/route/a/b/c", true},
		{"/telegram/route/*path", "/telegram/route", true},
		{"/telegram/route/*path", "/paper/route/a", false},
	}
	for _, c := range cases {
		if got := matchPattern(c.pattern, c.path); got != c.want {
			t.Errorf("matchPattern(%q, %q) = %v, want %v", c.pattern, c.path, got, c.want)
		}
	}
}

func TestPathHTMLHighlightsParams(t *testing.T) {
	got := string(pathHTML("/bilibili/user/video/:uid"))
	want := `/bilibili/user/video/<span class="p">:uid</span>`
	if got != want {
		t.Errorf("pathHTML = %q, want %q", got, want)
	}
}
