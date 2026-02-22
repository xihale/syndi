# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Tech Stack

- **Language**: Go 1.25.6
- **Framework**: gin-gonic/gin v1.10.0 (web framework)
- **HTML Parsing**: PuerkitoBio/goquery v1.11.0 (jQuery-like)
- **Caching**: hashicorp/golang-lru/v2 v2.0.7 (LRU cache)
- **Logging**: go.uber.org/zap v1.26.0 (structured logging, KDL format)
- **Config**: gopkg.in/yaml.v3 (YAML configuration files)
- **Testing**: stretchr/testify v1.9.0

## Project Structure

```
rsshub-go/
├── cmd/
│   └── server.go              # Main entry point, server bootstrap
├── internal/                  # Internal packages (not for external use)
│   ├── cache/handler.go       # Handler-level caching implementation (RECOMMENDED)
│   ├── client/                # HTTP client with retry, proxy, rate limiting
│   ├── middleware/            # Gin middleware (recovery, logger, headers, parameter)
│   ├── parser/                # HTML parsing utilities (goquery wrapper, content extraction)
│   └── routeutils/            # Route helper utilities (feed building, content cleaning)
├── pkg/                       # Public packages (for external use)
│   ├── cache/                 # Cache interface and MemoryCache implementation
│   ├── config/                # Configuration management from YAML files
│   ├── context/               # Request context wrapper (params, client, cache)
│   ├── docs/                  # Auto-generated documentation system
│   ├── logger/                # Zap logger initialization
│   ├── models/                # Core data structures (Feed, Item, Route, Category)
│   ├── registry/              # Route registry (singleton for route discovery)
│   ├── rss/                   # RSS/Atom XML generation
│   └── utils/date/            # Date parsing utilities
├── routes/                    # Route implementations (auto-registered via init())
│   ├── github/                # GitHub routes
│   ├── hackernews/            # HackerNews routes
│   ├── npm/                   # NPM routes
│   └── reddit/                # Reddit routes
├── examples/                  # Example implementations
├── docs/                      # Documentation (CACHING.md, MIGRATION.md)
├── config.yaml                # Configuration file (YAML)
└── Makefile                   # Build automation
```

## Common Development Commands

```bash
# Build
make build              # Build binary to build/rsshub-go
make run                # Run directly (go run ./cmd/server)

# Testing
make test               # Run all tests
make test-coverage      # Generate coverage report (coverage.html)
go test ./...           # Alternative test command
go test -v ./...        # Verbose tests

# Code Quality
make fmt                # Format code (go fmt + goimports)
make lint               # Run golangci-lint (if installed)

# Configuration
make install-config     # Install config to /etc/rsshub-go/ (requires sudo)

# Maintenance
make clean              # Remove build artifacts
make deps               # Download and tidy dependencies
make help               # Show all targets
```

## Architecture Overview

### Server Startup (`cmd/server.go`)

1. Load configuration from YAML file (`config.yaml` or `/etc/rsshub-go/config.yaml`)
2. Initialize zap logger with KDL format (human-readable structured logging)
3. Create cache instance (MemoryCache, Redis placeholder)
4. Create HTTP client with retry logic and rate limiting
5. Create Gin engine with middleware stack
6. Register documentation routes (`/docs`, `/api/routes`, etc.)
7. Register health check (`/status`)
8. Register all routes from global registry with handler-level caching
9. Start HTTP server with graceful shutdown on SIGINT/SIGTERM

**Log Format**: KDL (KDL Document Language) - human-readable structured logging
```
info "message" {
    timestamp="2026-02-18T19:29:48+08:00"
    file="/path/to/file.go:42"
    key="value"
    count=42
}
```

### Middleware Stack (order matters)

1. **Recovery** - Panic recovery
2. **Logger** - HTTP request logging
3. **Header** - CORS, security headers, Cache-Control

**Note**: Caching is done at the handler level, not middleware level.

### Route Registration System

Routes are auto-registered via `init()` functions in each route package:

```go
// In routes/github/trending.go
func init() {
    cacheTTL := 30 * time.Minute

    route := &models.Route{
        Path:         "/github/trending/:language",
        Name:         "GitHub Trending",
        Handler:      GitHubTrendingHandler,
        Parameters:   []models.Parameter{{Name: "language", Required: false}},
        CacheTTL:     &cacheTTL,
        // ... more metadata
    }

    if err := registry.GetRegistry().Register(route); err != nil {
        panic(err)
    }
}
```

Route packages are imported in `cmd/server.go` (lines 27-32) to trigger `init()`.

### Handler Signature

All route handlers have the signature:
```go
type HandlerFunc func(*ctxpkg.Context) (*models.Feed, error)
```

The `ctxpkg.Context` provides:
- `Param(key string)` - Path parameters
- `Client()` - HTTP client for making requests
- `Cache()` - Cache instance
- `Parent()` - Original context.Context

## Caching System

**Important**: Use handler-level caching (recommended), not middleware-level caching.

Handler-level caching (`internal/cache/handler.go`):
- `NewCachedHandler(cache, handler)` - Default 15 min TTL
- `NewCachedHandlerWithTTL(cache, handler, ttl)` - Custom TTL
- `Cached(cache, handler, opts)` - Full customization

Features:
- Per-route TTL control
- ETag support for 304 Not Modified
- X-Cache-Status header (HIT/MISS/ERROR)
- Error responses (4xx/5xx) not cached
- Custom key generation and conditional caching

See `docs/CACHING.md` for comprehensive caching documentation.

## HTTP Client (`internal/client/client.go`)

Features:
- Configurable timeout, user agent, max redirects
- Retry logic with exponential backoff
- Proxy support (configured in YAML)
- Rate limiting per host (token bucket)
- Cookie persistence
- Convenience methods: `GetJSON()`, `GetXML()`, `GetHTML()`

## Configuration

Configuration is loaded from YAML files (`pkg/config/config.go`):

**Config file search order**:
1. `RSSHUB_CONFIG` environment variable (if set)
2. `./config.yaml` (current directory)
3. `/etc/rsshub-go/config.yaml` (system-wide)

**Configuration structure** (see `config.yaml` for full example):
```yaml
server:
  port: "1200"              # Server port
  env: "production"         # Environment: production/development/test
  read_timeout: 30s         # Read timeout
  write_timeout: 30s        # Write timeout
  idle_timeout: 120s        # Idle timeout

cache:
  type: "memory"            # Cache type: memory
  ttl: 15m                  # Cache TTL
  memory_size: 10000        # Memory cache size (if type is memory)

client:
  user_agent: "RSSHub-Go/1.0"
  timeout: 30s
  max_redirects: 10
  proxy: ""                 # Proxy URL
  no_proxy: false           # Disable proxy

routes:
  disable_nsfw: false       # Filter NSFW routes

middleware:
  enable_cache: true        # Enable caching globally
  access_key: ""            # Access control key
  allow_origin: "*"         # CORS allowed origin
```

**Accessing config in code**:
```go
cfg, err := config.Load("")  // Auto-detect config file
// or
cfg := config.LoadOrPanic("/path/to/config.yaml")

// Access nested config with backward compatibility getters
port := cfg.GetPort()              // Returns cfg.Server.Port
ttl := cfg.GetCacheTTL()           // Returns cfg.Cache.TTL
```

**Route-specific configs**: Still use environment variables (e.g., `TWITTER_COOKIE`, `GITHUB_TOKEN`) accessed via `cfg.Get(key, defaultValue)`.

## Route Implementation Pattern

Example from `routes/github/trending.go`:

```go
package routes

import (
    ctxpkg "github.com/xihale/rsshub-go/pkg/context"
    "github.com/xihale/rsshub-go/pkg/models"
    "github.com/xihale/rsshub-go/pkg/registry"
    "github.com/xihale/rsshub-go/internal/routeutils"
)

func init() {
    cacheTTL := 30 * time.Minute

    route := &models.Route{
        Path:         "/github/trending/:language",
        Name:         "GitHub Trending",
        Handler:      GitHubTrendingHandler,
        Parameters: []models.Parameter{
            {Name: "language", Required: false, Description: "..."},
        },
        CacheTTL: &cacheTTL,
    }

    if err := registry.GetRegistry().Register(route); err != nil {
        panic(err)
    }
}

func GitHubTrendingHandler(c *ctxpkg.Context) (*models.Feed, error) {
    language := c.Param("language")

    // Fetch HTML
    doc, err := routeutils.GetHTML(ctx, c.Client(), url)
    if err != nil {
        return nil, err
    }

    // Build feed
    feed := routeutils.NewFeed(title, link, description)

    // Add items
    feed.Items = append(feed.Items, models.Item{
        Title:       "Item title",
        Link:        "https://example.com/item",
        GUID:        "unique-id",
        Description: "Item content",
        PubDate:     time.Now(),
    })

    return feed, nil
}
```

## Route Utilities (`internal/routeutils/`)

Helper functions for common route operations:
- `NewFeed(title, link, description)` - Create a new feed
- `NewItem(title, link, description, pubDate)` - Create a new item
- `GetHTML(ctx, client, url)` - Fetch and parse HTML
- `AddItem(feed, item)` - Add item to feed
- `SetCategories(item, categories...)` - Set item categories

## Documentation

Auto-generated documentation available at:
- HTML: `/docs` - List all routes
- HTML: `/docs/route?path=<path>` - Detail view
- JSON: `/api/routes` - All routes as JSON
- JSON: `/api/namespaces` - Namespace list
- JSON: `/api/categories` - Category list

See `docs/CACHING.md` for comprehensive caching guide.

## Testing

Run tests with `make test` or `go test ./...`.

Test files use `*_test.go` naming convention and `testing` package.

Configuration tests create temporary YAML files to test loading behavior.

## Important Files

- `cmd/server.go` - Main entry point, server setup
- `config.yaml` - Default configuration file
- `internal/cache/handler.go` - Handler-level caching (RECOMMENDED)
- `internal/client/client.go` - HTTP client implementation
- `pkg/models/feed.go` - Core data structures
- `pkg/registry/registry.go` - Route registration system
- `pkg/config/config.go` - Configuration management (YAML-based)

## Notes

- Routes are auto-discovered via `init()` functions - no manual registration needed
- Use handler-level caching, not middleware-level caching (see docs/CACHING.md)
- All route handlers return `(*models.Feed, error)`
- Use `routeutils` helpers for common operations
- Gin path parameters use `:param` syntax (e.g., `/github/repos/:owner/:repo`)
- Configuration uses YAML files, not environment variables (except route-specific secrets)
