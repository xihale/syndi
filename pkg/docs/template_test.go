package docs

import (
	"testing"

	"github.com/xihale/rsshub-go/pkg/registry"
)

func TestRouteAndIndexTemplatesRender(t *testing.T) {
	indexTmpl, routeTmpl := ParseTemplates()
	route := &RouteDoc{Path: "/zhihu/hot", Name: "知乎热榜", Description: "d", CurlExample: "curl x"}
	data := RoutePageData{
		Route:   route,
		Related: []*RouteDoc{},
		RouteEnvStatuses: []registry.EnvStatus{{
			Namespace: "zhihu", Key: "ZHIHU_COOKIES", Description: "d", Scope: "s", Configured: false,
		}},
	}
	if err := routeTmpl.Execute(testWriter{}, data); err != nil {
		t.Fatalf("route template error: %v", err)
	}
	if err := indexTmpl.Execute(testWriter{}, PageData{Title: "t", EnvStatuses: registry.AllEnvStatuses()}); err != nil {
		t.Fatalf("index template error: %v", err)
	}
}

type testWriter struct{}

func (testWriter) Write(p []byte) (int, error) { return len(p), nil }
