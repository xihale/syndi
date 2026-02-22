# Repository Guidelines

## Project Structure & Module Organization
- `cmd/` holds the server entry point (`server.go`) that wires middleware, routes, and docs.
- `internal/` contains implementation-only helpers (clients, middleware, parser, route utilities); `pkg/` exposes reusable components (config, cache, models, registry, rss, utils).
- `routes/` is the collection of auto-registered route packages; `examples/` and `docs/` host route illustrations and supporting guides like `docs/CACHING.md`.
- Configuration and scripts live at the repo root (`config.yaml`, `Makefile`, `scripts/`), while generated binaries go to `build/`.

## Build, Test, and Development Commands
- `make build` compiles `cmd/server.go` into `build/rsshub-go` using Go 1.25 modules.
- `make run` boots the server with `go run ./cmd/server`; rely on this for quick local iterations.
- `make fmt` (`go fmt` + `goimports`) and `make lint` (golangci-lint when installed) enforce formatting and linting.
- `make test`, `go test ./...`, or `go test -v ./...` run the codebase tests; `make test-coverage` writes `coverage.html`.
- `make install-config` copies `config.yaml` to `/etc/rsshub-go/` (sudo) for system-wide installs; `make clean` removes artifacts.

## Coding Style & Naming Conventions
- Follow idiomatic Go: tabs for indentation, exported identifiers start with uppercase, file-level comments on packages.
- Run `make fmt` before committing; keep `go.mod` tidy after adding dependencies (`go mod tidy` or `make deps`).
- Route handlers follow `func(ctxpkg.Context) (*models.Feed, error)` and live under `routes/<provider>/<route>.go`; configuration helpers stay under `pkg/config`.
- Name tests `*_test.go`; companion fixtures belong alongside their package.

## Testing Guidelines
- Tests live alongside implementation (`internal/`, `pkg/`, `routes/` etc.) and rely on `stretchr/testify`.
- Run `go test ./...` for full coverage, `make test-coverage` when updating metrics, and include failing test reproduction steps in PRs.
- Keep test helpers (e.g., fixtures, mock clients) in `_test.go` files and avoid network calls unless explicitly mocked.

## Commit & Pull Request Guidelines
- Commit messages use `type: short description` (e.g., `feat: add new caching helper`), mirroring recent history.
- PRs should describe what changed, reference related issues/routes, explain tests run, and attach relevant docs/screenshots when adding routes or UI-facing features.
- Tag reviewers early for architectural routes (new directories under `routes/` or middleware changes) so caches, docs, and server startup receive scrutiny.

## Configuration & Deployment Tips
- Config loads from `RSSHUB_CONFIG`, then `./config.yaml`, then `/etc/rsshub-go/config.yaml`; share overrides via environment variables when needed.
- Use `config.yaml` defaults to adjust server timeouts, cache TTLs, client settings, and middleware behavior before committing route changes.
