# Testing Guide

How to test routes and infrastructure changes in this repo.

## Command Overview

| Task | Command |
|---|---|
| All tests | `go test ./...` |
| Short mode (no network) | `go test -short ./...` |
| One package | `go test -v ./internal/routeutils` |
| Live route fetch tests | `LIVE=1 go test ./routes/<ns>/ -run Live -v` |
| Route metadata check | `make verify-routes-strict` |
| Live-verify all routes (slow, needs running server + network) | `make verify-all` |
| Coverage report | `make test-coverage` |
| Full CI parity | `make ci-local` |

## Test Layout

Tests live alongside implementation (`*_test.go`) and use `stretchr/testify`.

Two kinds of route tests coexist:

### 1. Offline unit tests

Parse logic against fixtures or inline HTML/JSON. Must pass with `-short` and never touch the network:

```go
func TestParseThing(t *testing.T) {
	// pure function / fixture parsing assertions
}
```

Run: `go test -short ./routes/<ns>/`

### 2. Live tests (`LIVE=1` gated)

End-to-end handler runs against real upstreams via `internal/testutil.RunHandler`, which builds a real HTTP context + client:

```go
func TestNsThingLive(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("set LIVE=1 to run live fetch test")
	}
	feed, err := testutil.RunHandler(NsThingHandler, map[string]string{"id": "example-id"})
	if err != nil { t.Fatal(err) }
	if len(feed.Items) == 0 { t.Fatal("expected items") }
	t.Logf("got %d items, first: %s", len(feed.Items), feed.Items[0].Title)
}
```

See [PORTING_GUIDE.md](./PORTING_GUIDE.md) for authoring conventions.

## Manual Verification Against a Running Server

```bash
make build && ./build/syndi          # default port 1200

curl -s "http://localhost:1200/<namespace>/<route>" | head -50
curl -s ".../feed" | xmllint --format -        # validate RSS XML
```

Inspect registered metadata through the JSON APIs:

```bash
curl -s "http://localhost:1200/api/routes" | python3 -m json.tool
curl -s "http://localhost:1200/api/routes/<full/path>" | python3 -m json.tool
curl -s "http://localhost:1200/api/categories" | python3 -m json.tool
```

Cache behavior smoke test (`X-Cache` header flips MISS -> HIT, ETag gives 304):

```bash
curl -I http://localhost:1200/github/trending/daily/go
curl -I http://localhost:1200/github/trending/daily/go   # X-Cache: HIT
```

## Debugging Tips

- Raw upstream payload: `curl "<upstream-url>" | head -100`
- Verbose request path: `curl -v http://localhost:1200/<route>`
- Selector experiments against live HTML: write a throwaway Go file using `internal/parser.LoadString` + `.Each/.Find`, run with `go run`.
- Empty feed? Check selector drift on upstream, anti-bot pages (see [DISGUISE.md](./DISGUISE.md)), and whether items lack title AND link (they are skipped by design).

## Checklist Before Shipping a Route

- [ ] `go test -short ./routes/<ns>/` passes offline
- [ ] `LIVE=1 go test ./routes/<ns>/ -run Live -v` returns items
- [ ] `gofmt -l routes/<ns>` prints nothing; `go vet ./routes/<ns>/...` clean
- [ ] `make verify-routes-strict` passes
- [ ] `go run scripts/generate-routes.go` produces no diff in `cmd/routes_gen.go`
