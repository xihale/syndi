# Route Porting Guide

How to add or port routes into this Go project. Read fully before writing code.

## Architecture

- Each site namespace = one Go package under `routes/<namespace>/`, package name **must be `routes`** (exception: `routes/test` uses `test`).
- Every namespace MUST have `routes/<ns>/routes.go` exporting:
  ```go
  var Routes = []routeutils.RouteSpec{ /* all specs */ }
  ```
  A directory without it is silently skipped by `scripts/generate-routes.go`.
- Registration is centralized: route package imports are generated into `cmd/routes_gen.go`, and each package's `Routes` slice is registered via `routeutils.MustRegisterRoutesWithBase`, using the package folder name as base path (`routes/github` -> `/github`).
- After adding or renaming namespaces, regenerate the bootstrap yourself:
  ```bash
  go run scripts/generate-routes.go   # also run automatically by make build / make run
  ```
  (In batch-porting sessions a coordinator may centralize this step instead.)

## RouteSpec conventions

```go
var nsThingRoute = routeutils.RouteSpec{
	Path:        "thing/:id",            // relative to namespace base, no leading slash, :param style
	Name:        "Site Thing",           // English, Title Case
	Example:     "site/thing/123",       // full path WITH namespace, exactly what you curl to verify
	Maintainers: []string{"xihale"},
	Description: "One sentence describing the feed",
	Categories:  []models.Category{{Name: "programming"}},
	Features:    models.Features{SupportRadar: true}, // SupportRadar only when obvious single source
	Parameters: []models.Parameter{
		routeutils.RequiredParam("id", "what it is"),
		routeutils.OptionalParam("limit", "default 20, max 100"),
	},
	CacheTTL: 30 * time.Minute,
	Handler:  NsThingHandler, // func(c *ctxpkg.Context) (*models.Feed, error)
}
```

Allowed category names (pick closest): `programming`, `study`, `new-media`, `picture`, `game`, `finance`, `science`, `technology`, `live`, `social-media`.

CacheTTL guidance: fast-changing news 15–30min; blogs/releases 1–6h; heavily rate-limited APIs (launchlibrary, NASA DEMO_KEY) **≥ 2h**; static lists 12–24h.

## Handler pattern

Prefer this pipeline shape:

1. Parse path/query parameters.
2. Fetch upstream payload.
3. Create feed with `routeutils.NewFeed`.
4. Map payload records to `*models.Item` and append.
5. Return feed.

```go
func NsThingHandler(c *ctxpkg.Context) (*models.Feed, error) {
	id := c.Param("id")
	limit := routeutils.ParsePositiveInt(c.QueryParam("limit"), 20, 100)
	ctx := c.Parent()

	var resp myResp
	if err := routeutils.GetJSON(ctx, c.Client(), url, &resp); err != nil {
		return nil, err
	}

	feed := routeutils.NewFeed(title, link, description)
	for _, r := range resp.Items {
		item := routeutils.NewItem(r.Title, r.Link, htmlDescription, r.PublishedAt)
		if item == nil || item.Title == "" { continue }
		routeutils.AddItem(feed, item)
	}
	return feed, nil
}
```

Helpers (`internal/routeutils`): `GetJSON(ctx, client, url, &target)`, `GetXML`, `GetHTML(ctx, client, url) (*parser.Document)`,
`GetFeed(ctx, client, url) (*models.Feed, error)` — parses native RSS2/Atom/RDF, ideal for feed-wrapper routes,
`NewFeed(title, link, desc)`, `NewItem(title, link, description, pubDate)` (returns nil-safe item),
`AddItem(feed, item)` (ignores nils), `SetItemAuthor(item, name, email, uri)`, `SetCategories(item, ...)`,
`AppendMappedItems(feed, slice, maxItems, mapper)` (fills GUID from Link when empty),
`ParsePositiveInt`, `ParseBool`, `ParseEnum`.
HTML parsing: `parser.Document` wraps goquery: `.Each(sel, fn)`, `.Find(sel)`, selection `.Text() .AttrOr(name, def) .Find(sel)`.

## Credentials & docs frontend

If a namespace depends on an environment variable (cookie, token, API key), declare it in the package `init()` so the docs frontend shows its live status:

```go
func init() {
	registry.RegisterNamespaceEnv("zhihu", registry.EnvRequirement{
		Key:         "ZHIHU_COOKIES",
		Description: "知乎登录 Cookie（至少包含 z_c0）。",
		Scope:       "部分路由（登录类）",
	})
}
```

The frontend (`/`) renders a CREDENTIALS panel with ✓/✗ state resolved from the live process environment; `/api/config` exposes the same as JSON. Only booleans are exposed — values are never echoed back.

## Hard quality rules

1. **Probe the upstream first.** Before writing Go, `curl` the exact endpoint you plan to hit and confirm the response shape and reachability from this machine. Design structs against the REAL payload.
2. Dates: parse with `dateutil.ParseDate(str)` (`github.com/xihale/syndi/pkg/utils/date`) or unmarshal RFC3339 directly into `time.Time`. Unix timestamps: `time.Unix(sec, 0)`. **If no date exists, leave PubDate zero. Never fake with time.Now().**
3. `item.Description` = article content only (may contain HTML). Put metadata (author, tags, stats) via helpers or appended `<br/>` lines.
4. GUID: set explicitly when upstream id exists (`item.GUID = id`). Otherwise NewItem defaults GUID to link.
5. Unique links per item; skip entries lacking title AND link.
6. Escape text going INTO HTML descriptions with `html.EscapeString` when it isn't markup.
7. Respect upstream ToS signals: if a probe gets 403/blocked/challenge pages, mark the route SKIP in your report (do not ship broken scrapers).
8. No secrets in code. Public/demo keys only where upstream officially documents them.
9. For sites that block default clients use the request disguise API — see [DISGUISE.md](./DISGUISE.md); keep transport behavior (retry/proxy/rate-limit) untouched.

## Verification (mandatory per route)

Use the shared live-test harness. Create `<ns>/<file>_test.go`:

```go
func TestNsThingLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(NsThingHandler, map[string]string{"id": "example-id-value"})
	if err != nil { t.Fatal(err) }
	if len(feed.Items) == 0 { t.Fatal("expected items") }
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}
```

`internal/testutil.RunHandler(handler, params)` builds a real HTTP context + client. Run (see [TESTING.md](./TESTING.md) for details):

```
gofmt -l routes/<ns>
go vet ./routes/<ns>/...
LIVE=1 go test ./routes/<ns>/ -run Live -v
```

All three must pass. If upstream is temporarily down, still ship structurally sound code but say so in your report.

Then run the metadata verifier:

```bash
make verify-routes            # warnings reported
make verify-routes-strict     # warnings fail the build (CI parity)
```

It checks: required core metadata (path/name/handler), path parameter metadata presence, duplicate parameter names, example hygiene (namespace prefix, no URLs/placeholders), parameter descriptions, duplicate categories, cache TTL outliers, and common quality issues (empty descriptions/examples, placeholder maintainers).

## Scaffolding

Use the scaffold command to create new route files quickly:

```bash
make new-route \
  NS=github \
  FILE=stars \
  ROUTE_PATH=stars/:owner \
  ROUTE_NAME="GitHub Stars" \
  EXAMPLE=github/stars/octocat
```

The scaffold also updates `routes/<namespace>/routes.go` and regenerates `cmd/routes_gen.go`.

## Report format (batch porting tasks)

Per route: `PATH | OK/SKIP/DEGRADED | items-in-live-test | notes`
Then: files created, anything unusual.
