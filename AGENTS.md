# Repository Guidelines

## Project Structure & Module Organization
- `cmd/` holds the server entry point (`server.go`) that wires middleware, routes, and docs.
- `internal/` contains implementation-only helpers (clients, middleware, parser, route utilities); `pkg/` exposes reusable components (config, cache, models, registry, rss, utils).
- `routes/` is the collection of auto-registered route packages; `examples/` and `docs/` host route illustrations and supporting guides like `docs/CACHING.md`.
- Configuration and scripts live at the repo root (`config.yaml`, `Makefile`, `scripts/`), while generated binaries go to `build/`.

## Build, Test, and Development Commands
- `make build` compiles `cmd/server.go` into `build/syndi` using Go 1.25 modules.
- `make run` boots the server with `go run ./cmd/server`; rely on this for quick local iterations.
- `make fmt` (`go fmt` + `goimports`) and `make lint` (golangci-lint when installed) enforce formatting and linting.
- `make test`, `go test ./...`, or `go test -v ./...` run the codebase tests; `make test-coverage` writes `coverage.html`.
- `make install-config` copies `config.yaml` to `/etc/syndi/` (sudo) for system-wide installs; `make clean` removes artifacts.

## Coding Style & Naming Conventions
- Follow idiomatic Go: tabs for indentation, exported identifiers start with uppercase, file-level comments on packages.
- Run `make fmt` before committing; keep `go.mod` tidy after adding dependencies (`go mod tidy` or `make deps`).
- Route handlers follow `func(ctxpkg.Context) (*models.Feed, error)` and live under `routes/<provider>/<route>.go`; configuration helpers stay under `pkg/config`.
- Name tests `*_test.go`; companion fixtures belong alongside their package.

## Testing Guidelines
- Tests live alongside implementation (`internal/`, `pkg/`, `routes/` etc.) and rely on `stretchr/testify`.
- Run `go test ./...` for full coverage, `make test-coverage` when updating metrics, and include failing test reproduction steps in PRs.
- Keep test helpers (e.g., fixtures, mock clients) in `_test.go` files and avoid network calls unless explicitly mocked.
- Route handlers may use live network tests guarded by the `LIVE` env var via `internal/testutil.RunHandler`; run them with `LIVE=1 go test ./routes/<ns>/ -run Live`.

## Adding or Modifying Routes
- Read `docs/PORTING_GUIDE.md` first: namespace package layout, `RouteSpec` metadata, date/description/GUID rules, and verification workflow.
- Every route package needs a `routes.go` exporting `Routes []routeutils.RouteSpec`; a directory without it is silently skipped by `scripts/generate-routes.go`.
- For sites that block default clients, prefer the request disguise API (`docs/DISGUISE.md`): e.g. `disguise.Chrome().Lang("zh-CN").GetHTML(ctx, c.Client(), url)` — keep transport behavior (retry/proxy/rate-limit) untouched.
- After adding namespaces, regenerate the bootstrap with `go run scripts/generate-routes.go`, then `make verify-routes-strict`; live-check examples with `make verify-all`.

## Commit & Pull Request Guidelines
- Commit messages use `type: short description` (e.g., `feat: add new caching helper`), mirroring recent history.
- PRs should describe what changed, reference related issues/routes, explain tests run, and attach relevant docs/screenshots when adding routes or UI-facing features.
- Tag reviewers early for architectural routes (new directories under `routes/` or middleware changes) so caches, docs, and server startup receive scrutiny.

## Configuration & Deployment Tips
- Config loads from `RSSHUB_CONFIG`, then `./config.yaml`, then `/etc/syndi/config.yaml`; share overrides via environment variables when needed.
- Use `config.yaml` defaults to adjust server timeouts, cache TTLs, client settings, and middleware behavior before committing route changes.
