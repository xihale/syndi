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
	// Title is required by the shared base template's <title> tag.
	Title            string
	Route            *RouteDoc
	Related          []*RouteDoc
	RouteEnvStatuses []CredStatus
}

// CredStatus groups the concrete cookies read from one env var.
type CredStatus struct {
	Key    string
	Fields []CredField
}

// CredField is one cookie and whether it is present in the env value.
type CredField struct {
	Name    string
	Present bool
}

const baseTemplate = `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="color-scheme" content="light dark">
    <title>{{.Title}} — RSSHub Go</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        :root {
    color-scheme: light dark;
    --accent: #e00000;
    --ok: #0a7d33;
    --bg: #ffffff;
    --fg: #111111;
    --fg-soft: #333333;
    --muted: #8a8a8a;
    --hairline: #e8e8e8;
    --rule: #111111;
    --code-bg: #f5f5f5;
    --hover-bg: #fafafa;
    --path: #0b57d0;

}
@media (prefers-color-scheme: dark) {
    :root {
        --accent: #ff5449;
        --ok: #4ade80;
        --bg: #111111;
        --fg: #ececec;
        --fg-soft: #c9c9c9;
        --muted: #8a8a8a;
        --hairline: #2a2a2a;
        --rule: #ececec;
        --code-bg: #1c1c1c;
        --hover-bg: #1a1a1a;
        --path: #7aa7ff;

    }
}
        body {
            font-family: "Helvetica Neue", Helvetica, Arial, "PingFang SC", "Microsoft YaHei", sans-serif;
            color: var(--fg);
            background: var(--bg);
            font-size: 15px;
            line-height: 1.6;
            -webkit-font-smoothing: antialiased;
        }
        .mono { font-family: "SF Mono", ui-monospace, Menlo, Consolas, monospace; }
        .container { max-width: 920px; margin: 0 auto; padding: 0 40px; }
        a { color: inherit; text-decoration: none; }

        header { padding: 64px 0 36px; border-bottom: 2px solid var(--rule); }
        header h1 { font-size: 62px; font-weight: 700; letter-spacing: -0.03em; line-height: 1; }
        header nav { margin-top: 20px; font-size: 11px; letter-spacing: 0.16em; text-transform: uppercase; color: var(--muted); }
        header nav a { margin-right: 28px; transition: color .12s; }
        header nav a:hover { color: var(--accent); }

        .label { font-size: 11px; font-weight: 700; letter-spacing: 0.16em; text-transform: uppercase; color: var(--muted); }

        .search-box { padding: 40px 0 4px; }
        .search-box input {
            width: 100%;
            border: none;
            border-bottom: 1px solid var(--rule);
            border-radius: 0;
            background: transparent;
            font: inherit;
            font-size: 19px;
            padding: 10px 0;
            outline: none;
        }
        .search-box input:focus { border-bottom: 1px solid var(--accent); }
        .search-box input::placeholder { color: #bbb; }

        .namespace { padding-top: 48px; }
        .ns-header {
            display: flex; justify-content: space-between; align-items: baseline;
            padding-bottom: 10px; border-bottom: 2px solid var(--rule);
        }
        .ns-header h2 { font-size: 24px; font-weight: 700; letter-spacing: -0.01em; }
        .count { font-size: 13px; color: var(--muted); }

        .route {
            display: grid;
            grid-template-columns: minmax(240px, 5fr) 4fr minmax(120px, 3fr);
            gap: 24px;
            align-items: baseline;
            padding: 13px 4px;
            border-bottom: 1px solid var(--hairline);
            cursor: pointer;
            transition: background .1s;
        }
        .route:hover { background: var(--hover-bg); }
        .r-path { font-family: "SF Mono", ui-monospace, Menlo, monospace; font-size: 13px; word-break: break-all; color: var(--path); }
        .r-path .p, .d-path .p { color: var(--accent); }
        .r-name { font-size: 15px; color: var(--fg-soft); }
        .r-meta {
            text-align: right;
            font-size: 11px;
            letter-spacing: 0.08em;
            text-transform: uppercase;
            white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
        }
        .r-meta .ttl { color: var(--muted); }

        /* ---- detail page ---- */
        .back { display: inline-block; margin: 40px 0 24px; font-size: 11px; letter-spacing: 0.16em; text-transform: uppercase; color: var(--muted); }
        .back:hover { color: var(--accent); }
        .d-title { font-size: 42px; font-weight: 700; letter-spacing: -0.02em; line-height: 1.15; }
        .d-path { font-family: "SF Mono", ui-monospace, Menlo, monospace; font-size: 15px; margin-top: 10px; word-break: break-all; color: var(--path); }
        .d-desc { margin-top: 20px; font-size: 16px; color: var(--fg-soft); max-width: 640px; }
        .d-cats { margin-top: 14px; font-size: 11px; letter-spacing: 0.12em; text-transform: uppercase; }
        .d-cats .ttl { color: var(--muted); }
        .route.off { opacity: 0.32; }
        .route.off:hover { opacity: 0.6; }
        ul.plain li.off { opacity: 0.32; }
        ul.plain li.off:hover { opacity: 0.6; }
        .cred { margin-top: 16px; }
        .cred-key { font-family: "SF Mono", ui-monospace, Menlo, monospace; font-size: 15px; font-weight: 700; }
        section { padding: 30px 0; border-bottom: 1px solid var(--hairline); }
        section .label { display: block; margin-bottom: 14px; }
        pre.example {
            background: var(--code-bg);
            padding: 16px 20px;
            font-family: "SF Mono", ui-monospace, Menlo, monospace;
            font-size: 13px;
            line-height: 1.7;
            overflow-x: auto;
            white-space: pre-wrap;
            word-break: break-all;
        }
        .kv { display: grid; grid-template-columns: 170px 1fr; gap: 10px 28px; font-size: 14px; }
        .kv dt { font-family: "SF Mono", ui-monospace, Menlo, monospace; font-size: 13px; font-weight: 600; }
        .kv dd { color: var(--fg-soft); }
        .kv dd.yes { color: var(--ok); font-weight: 600; }
        .kv dd.no { color: var(--accent); font-weight: 600; }
        .no { color: var(--accent); }
        .st { font-size: 11px; letter-spacing: 0.14em; text-transform: uppercase; }
        .kv dd code { font-family: "SF Mono", ui-monospace, Menlo, monospace; font-size: 13px; color: var(--accent); }
        .try-row { display: flex; gap: 16px; align-items: stretch; }
        .try-row input {
            flex: 1;
            border: none;
            border-bottom: 1px solid var(--rule);
            border-radius: 0;
            background: transparent;
            font-family: "SF Mono", ui-monospace, Menlo, monospace;
            font-size: 13px;
            padding: 10px 0;
            outline: none;
            color: var(--fg);
        }
        .try-row input:focus { border-bottom-color: var(--accent); }
        button.run {
            background: var(--fg);
            color: var(--bg);
            border: none;
            padding: 10px 20px;
            font: inherit;
            font-size: 11px;
            font-weight: 700;
            letter-spacing: 0.16em;
            cursor: pointer;
            transition: background .12s;
        }
        button.run:hover { background: var(--accent); }
        #tryOut { margin-top: 18px; max-height: 420px; overflow: auto; }
        ul.plain { list-style: none; }
        ul.plain li { padding: 10px 0; border-bottom: 1px solid var(--hairline); cursor: pointer; }
        ul.plain li:last-child { border-bottom: none; }
        ul.plain li:hover { background: var(--hover-bg); }
        ul.plain .r-path { margin-right: 16px; }

        footer {
            margin-top: 96px;
            padding: 26px 0 56px;
            border-top: 2px solid var(--rule);
            display: flex; justify-content: space-between;
            font-size: 12px; color: var(--muted);
        }
        @media (max-width: 720px) {
            .container { padding: 0 20px; }
            header { padding: 40px 0 28px; }
            header h1 { font-size: 40px; }
            .route { grid-template-columns: 1fr; gap: 4px; }
            .r-meta { text-align: left; }
            .kv { grid-template-columns: 1fr; gap: 2px 0; }
            .kv dd { margin-bottom: 10px; }
        }
    </style>
</head>
<body>
    <header>
        <div class="container">
            <h1>RSSHub Go</h1>
            <nav>
                <a href="/">routes</a>
                <a href="/api/routes">api</a>
                <a href="/robots.txt">robots.txt</a>
                <a href="https://github.com/xihale/rsshub-go">github</a>
            </nav>
        </div>
    </header>

    <div class="container">
        {{template "content" .}}
    </div>

    <footer>
        <div class="container" style="display:flex;justify-content:space-between;width:100%;">
            <span>RSSHub Go</span>
            <span>github.com/xihale/rsshub-go</span>
        </div>
    </footer>
</body>
</html>
`

const indexContent = `
{{define "content"}}
<div class="search-box">
    <input type="text" id="q" placeholder="搜索路由，按 / 聚焦" autocomplete="off" oninput="filterRoutes()">
</div>

<div id="routesContainer">
{{range .Namespaces}}
<div class="namespace" data-namespace="{{.Name}}">
    <div class="ns-header">
        <h2>{{.Name}}</h2>
        <span class="count">{{.RouteCount}}</span>
    </div>
    {{range .Routes}}
    <div class="route{{if .Unavailable}} off{{end}}" data-k="{{lower .Name}} {{lower .Description}} {{lower .Path}}"{{if .Unavailable}} title="缺少 {{range .MissingDeps}}{{.}} {{end}}配置"{{end}} onclick="location.href='/docs/route?path={{.Path}}'">
        <span class="r-path">{{pathHTML .Path}}</span>
        <span class="r-name">{{.Name}}</span>
        <span class="r-meta"><span class="ttl">{{.CacheTTL}}</span></span>
    </div>
    {{end}}
</div>
{{end}}
</div>

<script>
document.addEventListener('keydown', e => {
    const tag = (document.activeElement || {}).tagName;
    if (e.key === '/' && tag !== 'INPUT' && tag !== 'TEXTAREA') {
        e.preventDefault();
        document.getElementById('q').focus();
    } else if (e.key === 'Escape' && tag === 'INPUT') {
        document.getElementById('q').blur();
    }
});
function filterRoutes() {
    const q = document.getElementById('q').value.trim().toLowerCase();
    document.querySelectorAll('.namespace').forEach(ns => {
        let visible = 0;
        ns.querySelectorAll('.route').forEach(r => {
            const hit = !q || r.getAttribute('data-k').includes(q);
            r.style.display = hit ? '' : 'none';
            if (hit) visible++;
        });
        ns.style.display = visible > 0 ? '' : 'none';
    });
}
</script>
{{end}}
`

const routeContent = `
{{define "content"}}
<a class="back" href="/">&larr; 所有路由</a>

<h1 class="d-title">{{.Route.Name}}</h1>
<div class="d-path">{{pathHTML .Route.Path}}</div>
<p class="d-desc">{{.Route.Description}}</p>
<div class="d-cats"><span class="ttl">缓存 {{.Route.CacheTTL}}</span></div>

{{if .RouteEnvStatuses}}
<section>
    <span class="label">Credentials</span>
    {{range .RouteEnvStatuses}}
    <div class="cred">
        <div class="cred-key">{{.Key}}</div>
        <dl class="kv" style="margin-top:8px;">
            {{range .Fields}}
            <dt>{{.Name}}</dt>
            <dd class="st {{if .Present}}yes{{else}}no{{end}}">{{if .Present}}ok{{else}}missing{{end}}</dd>
            {{end}}
        </dl>
    </div>
    {{end}}
</section>
{{end}}

{{if .Route.ExampleURL}}
<section>
    <span class="label">Try</span>
    <form class="try-row" onsubmit="event.preventDefault();runTry()">
        <input id="tryPath" value="{{.Route.ExampleURL}}" spellcheck="false">
        <button class="run" type="submit">RUN</button>
    </form>
    <pre id="tryOut" class="example" hidden></pre>
</section>
{{end}}

{{if .Route.Parameters}}
<section>
    <span class="label">Params</span>
    <dl class="kv">
        {{range .Route.Parameters}}
        <dt>{{.Name}}{{if .Required}} <span class="no">*</span>{{end}}</dt>
        <dd>{{.Description}}</dd>
        {{end}}
    </dl>
</section>
{{end}}

{{if .Related}}
<section>
    <span class="label">Related</span>
    <ul class="plain">
    {{range .Related}}
        <li{{if .Unavailable}} class="off" title="缺少 {{range .MissingDeps}}{{.}} {{end}}配置"{{end}} onclick="location.href='/docs/route?path={{.Path}}'">
            <span class="r-path">{{pathHTML .Path}}</span><span class="r-name">{{.Name}}</span>
        </li>
    {{end}}
    </ul>
</section>
{{end}}

<script>
function runTry() {
    const out = document.getElementById('tryOut');
    const path = document.getElementById('tryPath').value.trim();
    out.hidden = false;
    out.textContent = 'loading…';
    fetch(path).then(async r => {
        const t = await r.text();
        out.textContent = r.status + ' ' + r.statusText + '\n\n' + t.slice(0, 8000);
    }).catch(e => { out.textContent = 'error: ' + e; });
}
</script>
{{end}}
`

// Template helpers
func join(strs []string, sep string) string {
	return strings.Join(strs, sep)
}

// pathHTML highlights parameter segments (:id, *path) inside a route path.
func pathHTML(path string) template.HTML {
	segs := strings.Split(path, "/")
	for i, s := range segs {
		if strings.HasPrefix(s, ":") || strings.HasPrefix(s, "*") {
			segs[i] = `<span class="p">` + template.HTMLEscapeString(s) + `</span>`
		}
	}
	return template.HTML(strings.Join(segs, "/"))
}

// ParseTemplates parses and returns the HTML templates
func ParseTemplates() (*template.Template, *template.Template) {
	funcMap := template.FuncMap{
		"lower":    strings.ToLower,
		"join":     join,
		"pathHTML": pathHTML,
	}

	// Index template
	indexTmpl := template.Must(template.New("index").Funcs(funcMap).Parse(baseTemplate))
	indexTmpl = template.Must(indexTmpl.Parse(indexContent))

	// Route detail template
	routeTmpl := template.Must(template.New("route").Funcs(funcMap).Parse(baseTemplate))
	routeTmpl = template.Must(routeTmpl.Parse(routeContent))

	return indexTmpl, routeTmpl
}
