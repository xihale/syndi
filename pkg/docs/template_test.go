package docs

import (
	"testing"
)

func TestRouteAndIndexTemplatesRender(t *testing.T) {
	indexTmpl, routeTmpl := ParseTemplates()

	if err := indexTmpl.Execute(testWriter{}, PageData{Title: "t"}); err != nil {
		t.Fatalf("index template error: %v", err)
	}

	route := &RouteDoc{
		Path:        "/zhihu/hot",
		Name:        "知乎热榜",
		Description: "d",
		CurlExample: "curl x",
		Categories:  []string{"social-media"},
	}
	data := RoutePageData{
		Title:   route.Name,
		Route:   route,
		Related: []*RouteDoc{},
		RouteEnvStatuses: []CredStatus{{
			Key:    "ZHIHU_COOKIES",
			Fields: []CredField{{Name: "z_c0", Present: false}},
		}},
	}
	if err := routeTmpl.Execute(testWriter{}, data); err != nil {
		t.Fatalf("route template error: %v", err)
	}

	data.RouteEnvStatuses[0].Fields[0].Present = true
	if err := routeTmpl.Execute(testWriter{}, data); err != nil {
		t.Fatalf("route template (configured) error: %v", err)
	}
}

type testWriter struct{}

func (testWriter) Write(p []byte) (int, error) { return len(p), nil }
