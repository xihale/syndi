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
	CrumbNS     string
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
    <title>{{.Title}} — Syndi</title>
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
        --rule: #3f3f3f;
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

        
        .label { font-size: 11px; font-weight: 700; letter-spacing: 0.16em; text-transform: uppercase; color: var(--muted); }

        .search-box { padding: 72px 0 4px; }
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
        }
        #routesContainer .namespace { margin-top: 24px; }
        #routesContainer .namespace:first-child { margin-top: 44px; }
        .ns-header h2 { font-size: 24px; font-weight: 700; letter-spacing: -0.01em; }
        .count { font-size: 13px; color: var(--muted); }

        .route {
            display: grid;
            grid-template-columns: minmax(240px, 5fr) 4fr minmax(120px, 3fr);
            gap: 24px;
            align-items: baseline;
            padding: 12px 4px;
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
        .crumbs { margin: 64px 0 28px; font-size: 11px; letter-spacing: 0.16em; text-transform: uppercase; color: var(--muted); }
        .crumbs a { transition: color .12s; }
        .crumbs a:hover { color: var(--accent); }
        .crumbs .sep { margin: 0 10px; }
        .crumbs .cur { color: var(--fg); }
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
        section { padding: 44px 0 0; }
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
        .try-row { display: flex; gap: 28px; align-items: flex-end; }
        .try-row input {
            flex: 1;
            border: none;
            border-bottom: 1px solid var(--rule);
            border-radius: 0;
            background: transparent;
            font-family: "SF Mono", ui-monospace, Menlo, Consolas, monospace;
            font-size: 13px;
            padding: 10px 0;
            outline: none;
            color: var(--fg);
        }
        .try-row input:focus { border-bottom-color: var(--accent); }
        button.run {
            background: transparent;
            color: var(--fg);
            border: 1px solid var(--fg);
            padding: 9px 26px;
            font: inherit;
            font-size: 11px;
            font-weight: 700;
            letter-spacing: 0.16em;
            cursor: pointer;
            transition: all .12s;
        }
        button.run:hover { border-color: var(--accent); color: var(--accent); }
        #tryStatus {
            margin-top: 22px;
            font-family: "SF Mono", ui-monospace, Menlo, Consolas, monospace;
            font-size: 11px;
            letter-spacing: 0.14em;
            text-transform: uppercase;
        }
        #tryStatus.yes { color: var(--ok); }
        #tryStatus.no { color: var(--accent); }
        #tryOut {
            margin-top: 8px;
            padding: 16px 0 4px;
            font-family: "SF Mono", ui-monospace, Menlo, Consolas, monospace;
            font-size: 12px;
            line-height: 1.8;
            max-height: 65vh;
            overflow: auto;
            white-space: pre;
            tab-size: 2;
        }
        ul.plain { list-style: none; }
        ul.plain li { padding: 9px 0; cursor: pointer; }
        ul.plain li:hover { background: var(--hover-bg); }
        ul.plain .r-path { margin-right: 16px; }

        footer {
            margin-top: 120px;
            padding: 0 0 56px;
            display: flex; justify-content: space-between; align-items: baseline;
            font-size: 12px; color: var(--muted);
        }
        footer nav { font-size: 11px; letter-spacing: 0.16em; text-transform: uppercase; }
        footer nav a { margin-left: 24px; transition: color .12s; }
        footer nav a:hover { color: var(--accent); }
        @media (max-width: 720px) {
            .container { padding: 0 20px; }
            .route { grid-template-columns: 1fr; gap: 4px; }
            .r-meta { text-align: left; }
            .kv { grid-template-columns: 1fr; gap: 2px 0; }
            .kv dd { margin-bottom: 10px; }
        }
    </style>
</head>
<body>
    <div class="container">
        {{template "content" .}}
    </div>

    <footer>
        <div class="container" style="display:flex;justify-content:space-between;width:100%;">
            <span>Syndi</span>
            <nav>
                <a href="/">routes</a>
                <a href="/rss">rss</a>
                <a href="/robots.txt">robots.txt</a>
                <a href="https://github.com/xihale/syndi">github</a>
            </nav>
        </div>
    </footer>
</body>
</html>
`

const indexContent = `
{{define "content"}}
{{if .CrumbNS}}<nav class="crumbs"><a href="/">index</a><span class="sep">/</span><span class="cur">{{.CrumbNS}}</span></nav>
{{end}}
<div class="search-box"{{if .CrumbNS}} style="padding-top:36px"{{end}}>
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
    <div class="route{{if .Unavailable}} off{{end}}" data-k="{{lower .Name}} {{lower .Description}} {{lower .Path}}"{{if .Unavailable}} title="缺少 {{range .MissingDeps}}{{.}} {{end}}配置"{{end}} onclick="location.href='{{.Path}}'">
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
<nav class="crumbs"><a href="/">index</a>{{with seg1 .Route.Path}}<span class="sep">/</span><a href="/{{.}}">{{.}}</a>{{end}}<span class="sep">/</span><span class="cur">{{.Route.Name}}</span></nav>

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
    <div id="tryStatus" hidden></div>
    <pre id="tryOut" hidden></pre>
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
        <li{{if .Unavailable}} class="off" title="缺少 {{range .MissingDeps}}{{.}} {{end}}配置"{{end}} onclick="location.href='{{.Path}}'">
            <span class="r-path">{{pathHTML .Path}}</span><span class="r-name">{{.Name}}</span>
        </li>
    {{end}}
    </ul>
</section>
{{end}}

<script>
function fmtXML(t) {
    t = t.replace(/^\s*<\?xml[^>]*\?>/, '').trimStart();
    if (!t.startsWith('<')) return t;
    const lines = t.replace(/></g, '>\n<').split('\n');
    let depth = 0;
    return lines.map(raw => {
        const l = raw.trim();
        if (/^<\//.test(l)) depth = Math.max(depth - 1, 0);
        const out = '  '.repeat(depth) + l;
        if (/^<[a-zA-Z]/.test(l) && !/<\/[a-zA-Z][\w:-]*>$/.test(l) && !/\/>$/.test(l)) depth++;
        return out;
    }).join('\n');
}
function runTry() {
    const out = document.getElementById('tryOut');
    const st = document.getElementById('tryStatus');
    const path = document.getElementById('tryPath').value.trim();
    out.hidden = false;
    st.hidden = false;
    st.className = '';
    st.textContent = 'LOADING';
    out.textContent = '';
    const started = performance.now();
    fetch(path).then(async r => {
        const ms = Math.round(performance.now() - started);
        st.className = r.ok ? 'yes' : 'no';
        st.textContent = r.status + ' ' + r.statusText + ' · ' + ms + 'MS';
        out.textContent = fmtXML(await r.text());
    }).catch(e => {
        st.className = 'no';
        st.textContent = 'FETCH FAILED';
        out.textContent = String(e);
    });
}
</script>
{{end}}
`

// Template helpers
func join(strs []string, sep string) string {
	return strings.Join(strs, sep)
}

// seg1 returns the first path segment (the namespace) of a route path.
func seg1(path string) string {
	parts := strings.SplitN(strings.Trim(path, "/"), "/", 2)
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return ""
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
		"seg1":     seg1,
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
