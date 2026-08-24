package docs

import (
	"html/template"
	"strings"

	"github.com/xihale/rsshub-go/pkg/registry"
)

// PageData represents the HTML page data
type PageData struct {
	Title       string
	Namespaces  []*NamespaceDoc
	TotalRoutes int
	Categories  []string
	EnvStatuses map[string][]registry.EnvStatus
}

// RoutePageData represents a single route page data
type RoutePageData struct {
	// Title is required by the shared base template's <title> tag.
	Title            string
	Route            *RouteDoc
	Related          []*RouteDoc
	RouteEnvStatuses []registry.EnvStatus
}

const baseTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - RSSHub Go</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', monospace;
            line-height: 1.5;
            color: #c5c8c6;
            background: #1d1f21;
            font-size: 14px;
        }
        .container {
            max-width: 900px;
            margin: 0 auto;
            padding: 20px 30px;
        }
        header {
            border-bottom: 1px solid #373b41;
            padding: 30px 0;
            margin-bottom: 30px;
        }
        header h1 {
            font-size: 1.5em;
            font-weight: 600;
            color: #f0f0f0;
            margin-bottom: 5px;
        }
        header p {
            color: #969896;
            font-size: 0.9em;
        }
        .search-box {
            margin-bottom: 30px;
        }
        .search-box input {
            width: 100%;
            padding: 10px;
            background: #282a2e;
            border: 1px solid #373b41;
            color: #c5c8c6;
            font-family: inherit;
            font-size: 14px;
        }
        .search-box input:focus {
            outline: none;
            border-color: #81a2be;
        }
        .stats {
            display: flex;
            gap: 30px;
            margin-bottom: 30px;
            padding-bottom: 20px;
            border-bottom: 1px solid #373b41;
        }
        .stat {
            color: #969896;
        }
        .stat strong {
            color: #81a2be;
            font-size: 1.2em;
        }
        .namespace {
            margin-bottom: 30px;
        }
        .namespace-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 10px 0;
            border-bottom: 1px solid #373b41;
            margin-bottom: 10px;
        }
        .namespace-header h2 {
            font-size: 1.1em;
            color: #f0f0f0;
            font-weight: 600;
        }
        .namespace-header .count {
            color: #969896;
            font-size: 0.9em;
        }
        .route {
            padding: 15px 0;
            border-bottom: 1px solid #282a2e;
        }
        .route:last-child {
            border-bottom: none;
        }
        .route-header {
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
            margin-bottom: 8px;
            gap: 15px;
        }
        .route-title h3 {
            font-size: 1em;
            color: #81a2be;
            font-weight: 600;
            margin-bottom: 3px;
        }
        .route-title .path {
            font-family: 'SF Mono', 'Monaco', 'Inconsolata', monospace;
            background: #282a2e;
            padding: 2px 6px;
            font-size: 0.85em;
            color: #b294bb;
        }
        .route-meta {
            display: flex;
            gap: 10px;
            flex-wrap: wrap;
            font-size: 0.9em;
        }
        .badge {
            padding: 2px 6px;
            font-size: 0.75em;
        }
        .badge-category { color: #8abeb7; }
        .badge-cache { color: #de935f; }
        .route-desc {
            color: #969896;
            margin: 8px 0;
            font-size: 0.9em;
        }
        .route-example {
            background: #282a2e;
            color: #b5bd68;
            padding: 10px;
            font-family: 'SF Mono', 'Monaco', 'Inconsolata', monospace;
            font-size: 0.8em;
            overflow-x: auto;
            margin-top: 8px;
        }
        .params {
            margin-top: 8px;
        }
        .param {
            display: inline-block;
            background: #282a2e;
            color: #cc6666;
            padding: 2px 6px;
            font-size: 0.8em;
            margin-right: 5px;
            margin-bottom: 3px;
        }
        .param .required {
            color: #de935f;
        }
        .back-link {
            display: inline-block;
            margin-bottom: 20px;
            color: #81a2be;
            text-decoration: none;
        }
        .back-link:hover {
            text-decoration: underline;
        }
        footer {
            text-align: center;
            padding: 30px 0;
            color: #969896;
            border-top: 1px solid #373b41;
            margin-top: 50px;
            font-size: 0.85em;
        }
        .query-params {
            background: #282a2e;
            padding: 12px;
            margin-top: 12px;
        }
        .query-params h4 {
            margin-bottom: 8px;
            color: #f0f0f0;
            font-size: 0.9em;
        }
        .query-param {
            margin-bottom: 6px;
            font-size: 0.85em;
            color: #969896;
        }
        .query-param code {
            background: #1d1f21;
            padding: 2px 5px;
            color: #b294bb;
        }
        .env-panel {
            background: #282a2e;
            border: 1px solid #373b41;
            padding: 15px;
            margin-bottom: 30px;
        }
        .env-panel h3 {
            color: #f0f0f0;
            font-size: 0.95em;
            margin-bottom: 10px;
        }
        .env-group {
            margin-bottom: 10px;
        }
        .env-group:last-child {
            margin-bottom: 0;
        }
        .env-item {
            display: flex;
            gap: 8px;
            align-items: baseline;
            padding: 3px 0;
            font-size: 0.85em;
        }
        .env-key {
            color: #b294bb;
            font-weight: 600;
        }
        .env-state-yes { color: #b5bd68; }
        .env-state-no { color: #cc6666; }
        .env-scope { color: #de935f; }
        .env-desc { color: #969896; }
        code {
            font-family: 'SF Mono', 'Monaco', 'Inconsolata', monospace;
        }
        a {
            color: #81a2be;
            text-decoration: none;
        }
        a:hover {
            text-decoration: underline;
        }
        @media (max-width: 768px) {
            .container {
                padding: 15px;
            }
            .route-header {
                flex-direction: column;
            }
            .stats {
                flex-direction: column;
                gap: 10px;
            }
        }
    </style>
</head>
<body>
    <header>
        <div class="container">
            <h1>RSSHub Go</h1>
            <p><a href="/">/</a> · <a href="/robots.txt">/robots.txt</a> · <a href="/api/routes">/api/routes</a></p>
        </div>
    </header>

    <div class="container">
        {{template "content" .}}
    </div>

    <footer>
        <p>github.com/xihale/rsshub-go</p>
    </footer>
</body>
</html>
`

const indexContent = `
{{define "content"}}
<div class="search-box">
    <input type="text" id="searchInput" placeholder="search routes..." onkeyup="filterRoutes()">
</div>

<div class="stats">
    <div class="stat"><strong>{{.TotalRoutes}}</strong> routes</div>
    <div class="stat"><strong>{{len .Namespaces}}</strong> namespaces</div>
    <div class="stat"><strong>{{len .Categories}}</strong> categories</div>
</div>

{{if .EnvStatuses}}
<div class="env-panel">
    <h3>CREDENTIALS / 配置状态</h3>
    {{range $ns, $statuses := .EnvStatuses}}
    <div class="env-group">
        {{range $statuses}}
        <div class="env-item">
            <span class="env-key">{{.Key}}</span>
            {{if .Configured}}<span class="env-state-yes">✓ 已配置</span>{{else}}<span class="env-state-no">✗ 未设置</span>{{end}}
            <span class="env-scope">[{{$ns}} · {{.Scope}}]</span>
            <span class="env-desc">{{.Description}}</span>
        </div>
        {{end}}
    </div>
    {{end}}
    <div class="env-desc" style="margin-top: 8px;">状态为服务进程运行时实时检测，仅显示是否配置，不回显值。</div>
</div>
{{end}}

<div id="routesContainer">
{{range .Namespaces}}
<div class="namespace" data-namespace="{{.Name}}">
    <div class="namespace-header">
        <h2>{{.Name}}</h2>
        <span class="count">{{.RouteCount}}</span>
    </div>
    <div class="route-list">
    {{range .Routes}}
    <div class="route" data-name="{{lower .Name}} {{lower .Description}} {{lower .Path}}">
        <div class="route-header">
            <div class="route-title">
                <h3>{{.Name}}</h3>
                <span class="path">{{.Path}}</span>
            </div>
            <div class="route-meta">
                {{range .Categories}}
                <span class="badge badge-category">{{.}}</span>
                {{end}}
                <span class="badge badge-cache">{{.CacheTTL}}</span>
            </div>
        </div>
        <p class="route-desc">{{.Description}}</p>
        {{if .Parameters}}
        <div class="params">
            {{range .Parameters}}
            <span class="param">{{.Name}}{{if .Required}} <span class="required">*</span>{{end}}</span>
            {{end}}
        </div>
        {{end}}
        <div class="route-example">{{.CurlExample}}</div>
    </div>
    {{end}}
    </div>
</div>
{{end}}
</div>

<script>
function filterRoutes() {
    const input = document.getElementById('searchInput');
    const filter = input.value.toLowerCase();
    const routes = document.querySelectorAll('.route');

    routes.forEach(route => {
        const searchText = route.getAttribute('data-name');
        if (searchText.includes(filter)) {
            route.style.display = '';
        } else {
            route.style.display = 'none';
        }
    });

    const namespaces = document.querySelectorAll('.namespace');
    namespaces.forEach(ns => {
        const visibleRoutes = ns.querySelectorAll('.route[style=""]');
        if (visibleRoutes.length === 0 && filter !== '') {
            const allHidden = Array.from(ns.querySelectorAll('.route')).every(r => r.style.display === 'none');
            ns.style.display = allHidden ? 'none' : '';
        } else {
            ns.style.display = '';
        }
    });
}
</script>
{{end}}
`

const routeContent = `
{{define "content"}}
<a href="/docs" class="back-link">← back</a>

<div class="namespace">
    <div class="namespace-header">
        <h2>{{.Route.Name}}</h2>
        <div class="route-meta">
            {{range .Route.Categories}}
            <span class="badge badge-category">{{.}}</span>
            {{end}}
            <span class="badge badge-cache">{{.Route.CacheTTL}}</span>
        </div>
    </div>
    <div class="route">
        <p class="route-desc" style="font-size: 1em; margin-bottom: 20px;">{{.Route.Description}}</p>

        {{if .RouteEnvStatuses}}
        <div class="env-panel" style="margin-bottom: 20px;">
            <h3>CREDENTIALS / 本路由凭据状态</h3>
            {{range .RouteEnvStatuses}}
            <div class="env-item">
                <span class="env-key">{{.Key}}</span>
                {{if .Configured}}<span class="env-state-yes">✓ 已配置</span>{{else}}<span class="env-state-no">✗ 未设置（相关路由将报错）</span>{{end}}
                <span class="env-scope">[{{.Scope}}]</span>
                <span class="env-desc">{{.Description}}</span>
            </div>
            {{end}}
        </div>
        {{end}}

        <h3 style="margin-bottom: 10px; color: #f0f0f0; font-size: 0.95em;">PATH</h3>
        <div class="route-example" style="margin-bottom: 20px;">{{.Route.Path}}</div>

        {{if .Route.Parameters}}
        <h3 style="margin-bottom: 10px; color: #f0f0f0; font-size: 0.95em;">PARAMS</h3>
        <div class="params" style="margin-bottom: 20px;">
            {{range .Route.Parameters}}
            <span class="param">{{.Name}}{{if .Required}} <span class="required">*</span>{{end}} - {{.Description}}</span>
            {{end}}
        </div>
        {{end}}

        <h3 style="margin-bottom: 10px; color: #f0f0f0; font-size: 0.95em;">EXAMPLE</h3>
        <div class="route-example" style="margin-bottom: 20px;">{{.Route.CurlExample}}</div>

        {{if .Route.QueryParams}}
        <div class="query-params">
            <h4>QUERY PARAMS</h4>
            {{range .Route.QueryParams}}
            <div class="query-param">
                <strong>{{.Name}}:</strong> {{.Description}}<br>
                <code>{{.Example}}</code>
            </div>
            {{end}}
        </div>
        {{end}}

        {{if .Route.Features}}
        <div style="margin-top: 20px;">
            {{if .Route.Features.SupportRadar}}<span style="color: #b5bd68; font-size: 0.85em;">✓ radar</span>{{end}}
            {{if .Route.Features.AntiCrawler}}<span style="color: #b5bd68; font-size: 0.85em;">✓ anti-crawler</span>{{end}}
        </div>
        {{end}}
    </div>
</div>

{{if .Related}}
<h3 style="margin-top: 30px; margin-bottom: 15px; color: #f0f0f0; font-size: 0.95em;">RELATED</h3>
<div class="route-list">
{{range .Related}}
<div class="route" style="cursor: pointer;" onclick="window.location.href='/docs/route?path={{.Path}}'">
    <div class="route-header">
        <div class="route-title">
            <h3>{{.Name}}</h3>
            <span class="path">{{.Path}}</span>
        </div>
    </div>
    <p class="route-desc">{{.Description}}</p>
</div>
{{end}}
</div>
{{end}}
{{end}}
`

// Template helpers
func join(strs []string, sep string) string {
	return strings.Join(strs, sep)
}

// ParseTemplates parses and returns the HTML templates
func ParseTemplates() (*template.Template, *template.Template) {
	funcMap := template.FuncMap{
		"lower": strings.ToLower,
		"join":  join,
	}

	// Index template
	indexTmpl := template.Must(template.New("index").Funcs(funcMap).Parse(baseTemplate))
	indexTmpl = template.Must(indexTmpl.Parse(indexContent))

	// Route detail template
	routeTmpl := template.Must(template.New("route").Funcs(funcMap).Parse(baseTemplate))
	routeTmpl = template.Must(routeTmpl.Parse(routeContent))

	return indexTmpl, routeTmpl
}
