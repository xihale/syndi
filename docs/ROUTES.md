# Route Architecture

This document describes the recommended route pattern for this repository.

## Goals

- Keep route metadata declarative and centralized.
- Keep handlers small and focused.
- Avoid repeated parsing, registration, and append boilerplate.

## Recommended File Layout

Each route file under `routes/<namespace>/` should follow this structure:

1. A package-level `routeutils.RouteSpec` variable holding metadata (path is relative to the namespace).
2. A package-level `Routes` slice listing all specs in the package (auto-registered by generated code).
3. A handler function implementing the route logic.
4. Optional pure helper functions near the handler for parsing/mapping.

## Core Helpers

- `routeutils.RouteSpec`: route metadata primitive (paths are relative to the namespace folder).
- `routeutils.MustRegisterRoutesWithBase`: used by generated bootstrap to register all specs.
- `routeutils.RequiredParam` / `routeutils.OptionalParam`: standardize param metadata.
- `routeutils.ParsePositiveInt`, `routeutils.ParseBool`, `routeutils.ParseEnum`: normalize query parsing.
- `routeutils.AppendMappedItems`: map source payloads to feed items with optional limit handling.
- Item defaults: `routeutils.NewItem`, `NewItemWithOptions`, `AddItem/AddItems`, and `AppendMappedItems` will fill `GUID` from `Link` when `GUID` is empty.

## Handler Pipeline Style

Prefer this shape:

1. Parse path/query parameters.
2. Fetch upstream payload.
3. Create feed with `routeutils.NewFeed`.
4. Map payload records to `*models.Item` and append with `AppendMappedItems`.
5. Return feed.

This keeps handlers easy to scan and test.

## Route Import Generation

Route package imports are generated into `cmd/routes_gen.go`.
The generated `registerRoutePackages()` function registers each package's `Routes` slice via
`routeutils.MustRegisterRoutesWithBase`, using the package folder name as the base path
(e.g., `routes/github` -> `/github`).
Only directories containing `routes.go` are treated as route packages, so helper-only folders
won't be registered.

- Generate manually: `go run scripts/generate-routes.go`
- Auto-generate during build/run: `make build`, `make run`

## Metadata Verification

Run route metadata checks:

```bash
make verify-routes
```

The verifier checks:

- required core metadata (path/name/handler)
- path parameter metadata presence
- duplicate parameter names
- example hygiene (namespace prefix, no URLs/placeholders)
- parameter descriptions
- duplicate categories
- duplicate path parameters
- cache TTL outliers (too short/long)
- common quality issues (empty descriptions/examples, placeholder maintainers)

Use strict mode to fail on warnings too:

```bash
make verify-routes-strict
# or:
go run ./scripts/verify-routes --strict
```

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
