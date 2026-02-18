package docs

import (
	"html/template"
	"strings"
)

// PageData represents the HTML page data
type PageData struct {
	Title       string
	Namespaces  []*NamespaceDoc
	TotalRoutes int
	Categories  []string
}

// RoutePageData represents a single route page data
type RoutePageData struct {
	Route       *RouteDoc
	Related     []*RouteDoc
}

const baseTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - RSSHub Go Documentation</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            background: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px 0;
            margin-bottom: 30px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        header h1 {
            font-size: 2.5em;
            margin-bottom: 10px;
        }
        header p {
            font-size: 1.1em;
            opacity: 0.9;
        }
        .search-box {
            background: white;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 30px;
            box-shadow: 0 2px 5px rgba(0,0,0,0.05);
        }
        .search-box input {
            width: 100%;
            padding: 12px 15px;
            border: 2px solid #e0e0e0;
            border-radius: 5px;
            font-size: 16px;
            transition: border-color 0.3s;
        }
        .search-box input:focus {
            outline: none;
            border-color: #667eea;
        }
        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .stat-card {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 5px rgba(0,0,0,0.05);
            text-align: center;
        }
        .stat-card h3 {
            font-size: 2em;
            color: #667eea;
            margin-bottom: 5px;
        }
        .stat-card p {
            color: #666;
            font-size: 0.9em;
        }
        .namespace {
            background: white;
            border-radius: 8px;
            margin-bottom: 20px;
            overflow: hidden;
            box-shadow: 0 2px 5px rgba(0,0,0,0.05);
        }
        .namespace-header {
            background: #f8f9fa;
            padding: 15px 20px;
            border-bottom: 2px solid #e0e0e0;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .namespace-header h2 {
            font-size: 1.3em;
            color: #333;
        }
        .namespace-header .count {
            background: #667eea;
            color: white;
            padding: 5px 12px;
            border-radius: 20px;
            font-size: 0.85em;
        }
        .route-list {
            padding: 0;
        }
        .route {
            padding: 20px;
            border-bottom: 1px solid #f0f0f0;
            transition: background 0.2s;
        }
        .route:last-child {
            border-bottom: none;
        }
        .route:hover {
            background: #f8f9fa;
        }
        .route-header {
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
            margin-bottom: 10px;
            flex-wrap: wrap;
            gap: 10px;
        }
        .route-title h3 {
            font-size: 1.1em;
            color: #667eea;
            margin-bottom: 5px;
        }
        .route-title .path {
            font-family: "Courier New", monospace;
            background: #f0f0f0;
            padding: 3px 8px;
            border-radius: 4px;
            font-size: 0.9em;
            color: #e83e8c;
        }
        .route-meta {
            display: flex;
            gap: 10px;
            flex-wrap: wrap;
        }
        .badge {
            padding: 3px 10px;
            border-radius: 12px;
            font-size: 0.75em;
            font-weight: 600;
        }
        .badge-category { background: #e3f2fd; color: #1976d2; }
        .badge-cache { background: #f3e5f5; color: #7b1fa2; }
        .route-desc {
            color: #666;
            margin-bottom: 10px;
        }
        .route-example {
            background: #263238;
            color: #aed581;
            padding: 10px 15px;
            border-radius: 5px;
            font-family: "Courier New", monospace;
            font-size: 0.85em;
            overflow-x: auto;
            margin-top: 10px;
        }
        .params {
            margin-top: 10px;
        }
        .param {
            display: inline-block;
            background: #fff3e0;
            color: #f57c00;
            padding: 3px 8px;
            border-radius: 4px;
            font-size: 0.85em;
            margin-right: 5px;
            margin-bottom: 5px;
        }
        .param .required {
            color: #d32f2f;
            font-weight: bold;
        }
        .back-link {
            display: inline-block;
            margin-bottom: 20px;
            color: #667eea;
            text-decoration: none;
        }
        .back-link:hover {
            text-decoration: underline;
        }
        footer {
            text-align: center;
            padding: 30px 0;
            color: #666;
            margin-top: 50px;
        }
        .query-params {
            background: #f8f9fa;
            padding: 15px;
            border-radius: 5px;
            margin-top: 15px;
        }
        .query-params h4 {
            margin-bottom: 10px;
            color: #333;
        }
        .query-param {
            margin-bottom: 8px;
            font-size: 0.9em;
        }
        .query-param code {
            background: #fff;
            padding: 2px 6px;
            border-radius: 3px;
            color: #e83e8c;
        }
        @media (max-width: 768px) {
            .route-header {
                flex-direction: column;
            }
            .stats {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <header>
        <div class="container">
            <h1>🚀 RSSHub Go</h1>
            <p>Lightweight, fast RSS feed generation service</p>
        </div>
    </header>

    <div class="container">
        {{template "content" .}}
    </div>

    <footer>
        <p>Powered by <strong>RSSHub Go</strong> | Generated from route metadata</p>
    </footer>
</body>
</html>
`

const indexContent = `
{{define "content"}}
<div class="search-box">
    <input type="text" id="searchInput" placeholder="🔍 Search routes by name, description, or path..." onkeyup="filterRoutes()">
</div>

<div class="stats">
    <div class="stat-card">
        <h3>{{.TotalRoutes}}</h3>
        <p>Total Routes</p>
    </div>
    <div class="stat-card">
        <h3>{{len .Namespaces}}</h3>
        <p>Namespaces</p>
    </div>
    <div class="stat-card">
        <h3>{{len .Categories}}</h3>
        <p>Categories</p>
    </div>
</div>

<div id="routesContainer">
{{range .Namespaces}}
<div class="namespace" data-namespace="{{.Name}}">
    <div class="namespace-header">
        <h2>{{.Name}}</h2>
        <span class="count">{{.RouteCount}} routes</span>
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
                <span class="badge badge-cache">⏱ {{.CacheTTL}}</span>
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

    // Hide empty namespaces
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

// Make lower available to template
function lower(s) {
    return s.toLowerCase();
}
</script>
{{end}}
`

const routeContent = `
{{define "content"}}
<a href="/docs" class="back-link">← Back to all routes</a>

<div class="namespace">
    <div class="namespace-header">
        <h2>{{.Route.Name}}</h2>
        <div class="route-meta">
            {{range .Route.Categories}}
            <span class="badge badge-category">{{.}}</span>
            {{end}}
            <span class="badge badge-cache">⏱ {{.Route.CacheTTL}}</span>
        </div>
    </div>
    <div class="route">
        <p class="route-desc" style="font-size: 1.1em; margin-bottom: 20px;">{{.Route.Description}}</p>

        <h3 style="margin-bottom: 10px;">📍 Route Path</h3>
        <div class="route-example" style="margin-bottom: 20px;">{{.Route.Path}}</div>

        {{if .Route.Parameters}}
        <h3 style="margin-bottom: 10px;">📝 Path Parameters</h3>
        <div class="params" style="margin-bottom: 20px;">
            {{range .Route.Parameters}}
            <span class="param">{{.Name}}{{if .Required}} <span class="required">*</span>{{end}} - {{.Description}}</span>
            {{end}}
        </div>
        {{end}}

        <h3 style="margin-bottom: 10px;">🚀 Example Usage</h3>
        <div class="route-example" style="margin-bottom: 20px;">{{.Route.CurlExample}}</div>

        {{if .Route.QueryParams}}
        <div class="query-params">
            <h4>❓ Query Parameters</h4>
            {{range .Route.QueryParams}}
            <div class="query-param">
                <strong>{{.Name}}:</strong> {{.Description}}<br>
                <code>{{.Example}}</code>
            </div>
            {{end}}
        </div>
        {{end}}

        {{if .Route.Features}}
        <h3 style="margin-top: 20px; margin-bottom: 10px;">⚡ Features</h3>
        <ul style="margin-left: 20px;">
            {{if .Route.Features.SupportRadar}}<li>✅ Supports Radar (WebSub)</li>{{end}}
            {{if .Route.Features.AntiCrawler}}<li>🛡️ Anti-crawler handling</li>{{end}}
        </ul>
        {{end}}

        {{if .Route.Maintainers}}
        <h3 style="margin-top: 20px; margin-bottom: 10px;">👥 Maintainers</h3>
        <p>{{join .Route.Maintainers ", "}}</p>
        {{end}}
    </div>
</div>

{{if .Related}}
<h3 style="margin-top: 30px; margin-bottom: 15px;">🔗 Related Routes</h3>
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
