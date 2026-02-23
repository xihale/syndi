# RSSHub Go - RSS feed generation framework

A high-performance RSS feed generation framework written in Go, inspired by RSSHub.

## Features

- 🚀 **High Performance**: Built on Go's concurrency model
- 🎯 **Easy to Use**: Simple API for creating RSS routes
- 📦 **Extensible**: Plugin-based architecture
- 🔧 **Flexible**: Support for HTML scraping, JSON APIs, XML feeds
- ⚡ **Production Ready**: Built-in caching, rate limiting, retries
- 🛡 **Type Safe**: Full Go types and interfaces
- 📝 **Configuration File**: YAML-based configuration

## Quick Start

```bash
# Build
make build

# Install configuration file (optional, requires sudo)
sudo make install-config

# Run
./build/rsshub-go
```

The server will look for `config.yaml` in the following order:
1. `./config.yaml` (current directory)
2. `/etc/rsshub-go/config.yaml` (system-wide)

You can also specify a custom config file path via the `RSSHUB_CONFIG` environment variable:

```bash
export RSSHUB_CONFIG=/path/to/custom-config.yaml
./build/rsshub-go
```

## Configuration

Configuration is done via a YAML file. Here's an example:

```yaml
# Server settings
server:
  port: "1200"
  env: "production"
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s

# Cache settings
cache:
  type: "memory"
  ttl: 15m
  memory_size: 10000

# HTTP client settings
client:
  user_agent: "RSSHub-Go/1.0"
  timeout: 30s
  max_redirects: 10
  proxy: ""
  no_proxy: false

# Route settings
routes:
  disable_nsfw: false

# Middleware settings
middleware:
  enable_cache: true
  access_key: ""
  allow_origin: "*"
```

See `config.yaml` for the default configuration with all options documented.
For proxy and redirect behavior details (including precedence), see `docs/CLIENT_CONFIG.md`.

## Development

```bash
# Run tests
make test

# Verify route metadata
make verify-routes

# Verify route metadata strictly (warnings fail)
make verify-routes-strict

# Run with coverage
make test-coverage

# Format code
make fmt

# Lint code
make lint

# Run directly (for development)
make run
```

## Project Structure

```
rsshub-go/
├── cmd/
│   ├── server.go           # Main entry point
│   └── routes_gen.go       # Auto-generated route imports
├── internal/
│   ├── cache/              # Handler-level caching
│   ├── client/             # HTTP client with retry, proxy
│   ├── middleware/         # Gin middleware
│   ├── parser/             # HTML parsing utilities
│   └── routeutils/         # Route helper utilities
├── pkg/
│   ├── cache/              # Cache interface and implementations
│   ├── config/             # Configuration management
│   ├── models/             # Data structures (Feed, Item, Route)
│   ├── registry/           # Route registry
│   └── rss/                # RSS/Atom XML generation
├── routes/                 # Route implementations
│   ├── github/
│   ├── hackernews/
│   └── ...
├── config.yaml             # Configuration file
└── Makefile                # Build automation
```

## Adding Routes

Routes are registered automatically from each package's `Routes` slice.
A directory is treated as a route package only when it contains `routes.go`.
The generated bootstrap in `cmd/routes_gen.go` registers every package using its folder
name as the base path.

Create a new route scaffold:

```bash
make new-route \
  NS=github \
  FILE=stars \
  ROUTE_PATH=stars/:owner \
  ROUTE_NAME="GitHub Stars" \
  EXAMPLE=github/stars/octocat
```

This creates:
- `routes/<namespace>/<file>.go`
- `routes/<namespace>/<file>_test.go`
- updates `routes/<namespace>/routes.go`
- and regenerates `cmd/routes_gen.go`

Recommended route structure:
- Declare metadata once with `routeutils.RouteSpec` in a package-level variable.
- Keep `RouteSpec.Path` relative to the namespace folder (omit `/github/` prefix).
- Expose a package-level `Routes` slice listing all specs for auto-registration.
- Keep handler logic focused on `fetch -> map -> build feed`.
- Prefer shared helpers like `routeutils.ParsePositiveInt`, `routeutils.ParseEnum`, and `routeutils.AppendMappedItems`.
- Route utils will auto-fill `Item.GUID` from `Item.Link` when `GUID` is empty.
- Run `make verify-routes` before commit to catch metadata/path-parameter mismatches.

## Documentation

- `/docs` - Interactive route documentation (when server is running)
- `docs/CACHING.md` - Comprehensive caching guide
- `docs/CLIENT_CONFIG.md` - HTTP client config behavior and precedence
- `docs/ROUTES.md` - Route architecture and scaffolding workflow
- `CLAUDE.md` - Developer guide

## License

AGPL-3.0
