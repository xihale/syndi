# Route Porting Guide (for batch contributors)

How to port RSSHub TypeScript routes to this Go project. Read fully before writing code.

## Architecture

- Each site namespace = one Go package under `routes/<namespace>/`, package name **must be `routes`** (exception: `routes/test` uses `test`).
- Every namespace MUST have `routes/<ns>/routes.go` exporting:
  ```go
  var Routes = []routeutils.RouteSpec{ /* all specs */ }
  ```
- Registration is centralized: `scripts/generate-routes.go` regenerates `cmd/routes_gen.go`.
  **DO NOT run generate-routes.go, DO NOT touch `cmd/` or any directory outside your assigned namespaces.** The coordinator regenerates centrally after review.

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
`AppendMappedItems(feed, slice, maxItems, mapper)`, `ParsePositiveInt`, `ParseBool`, `ParseEnum`.
HTML parsing: `parser.Document` wraps goquery: `.Each(sel, fn)`, `.Find(sel)`, selection `.Text() .AttrOr(name, def) .Find(sel)`.

## Hard quality rules

1. **Probe the upstream first.** Before writing Go, `curl` the exact endpoint you plan to hit and confirm the response shape and reachability from this machine. Design structs against the REAL payload.
2. Dates: parse with `dateutil.ParseDate(str)` (`github.com/xihale/rsshub-go/pkg/utils/date`) or unmarshal RFC3339 directly into `time.Time`. Unix timestamps: `time.Unix(sec, 0)`. **If no date exists, leave PubDate zero. Never fake with time.Now().**
3. `item.Description` = article content only (may contain HTML). Put metadata (author, tags, stats) via helpers or appended `<br/>` lines.
4. GUID: set explicitly when upstream id exists (`item.GUID = id`). Otherwise NewItem defaults GUID to link.
5. Unique links per item; skip entries lacking title AND link.
6. Escape text going INTO HTML descriptions with `html.EscapeString` when it isn't markup.
7. Respect upstream ToS signals: if a probe gets 403/blocked/challenge pages, mark the route SKIP in your report (do not ship broken scrapers).
8. No secrets in code. Public/demo keys only where upstream officially documents them.

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

`internal/testutil.RunHandler(handler, params)` builds a real HTTP context + client. Run:

```
gofmt -l routes/<ns>
go vet ./routes/<ns>/...
LIVE=1 go test ./routes/<ns>/ -run Live -v
```

All three must pass. If upstream is temporarily down, still ship structurally sound code but say so in your report.

## Report format (end of task)

Per route: `PATH | OK/SKIP/DEGRADED | items-in-live-test | notes`
Then: files created, anything unusual.
