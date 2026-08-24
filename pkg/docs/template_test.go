package docs

import (
	"testing"

	"github.com/xihale/rsshub-go/pkg/registry"
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
		RouteEnvStatuses: []registry.EnvStatus{{
			Namespace:   "zhihu",
			Key:         "ZHIHU_COOKIES",
			Description: "配置后解锁需登录的知乎路由",
			Scope:       "部分路由（登录类）",
			Configured:  false,
			Fields:      []registry.EnvField{{Name: "z_c0", Note: "n"}},
		}},
	}
	if err := routeTmpl.Execute(testWriter{}, data); err != nil {
		t.Fatalf("route template error: %v", err)
	}

	data.RouteEnvStatuses[0].Configured = true
	if err := routeTmpl.Execute(testWriter{}, data); err != nil {
		t.Fatalf("route template (configured) error: %v", err)
	}
}

type testWriter struct{}

func (testWriter) Write(p []byte) (int, error) { return len(p), nil }
